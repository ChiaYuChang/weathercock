package ollama

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/ollama/ollama/api"
)

var (
	ErrNoBaseURL             = errors.New("base URL cannot be empty")
	ErrNoDefaultModel        = errors.New("no default model")
	ErrIncompleteResponse    = errors.New("ollama request failed: response was incomplete")
	ErrCanNotConnectToServer = errors.New("can not connect to server")
	ErrModelNotFount         = errors.New("could not retrieve model from ollama API")
	ErrModelNotSupport       = errors.New("model not support")
	ErrInvalidOptionsType    = errors.New("invalid options type")
)

var (
	Parallel            = min(runtime.NumCPU(), 3)
	MaxRetries          = 4
	MaxRetryWaitingTime = 10 * time.Second
)

// Client implements the llm.LLM interface for interacting with the Ollama service.
type Client struct {
	*llm.BaseClient
	OllamaAPI *api.Client
}

// builder is used to construct an Ollama Client using the functional options pattern.
// It holds the configuration parameters needed to initialize the client.
type builder struct {
	URL          *url.URL
	Client       *http.Client
	Models       map[string]llm.Model
	DefaultGen   string
	DefaultEmbed string
}

type OllamaEmbedReq struct {
	Index int
	Req   *api.EmbeddingRequest
}

type OllamaEmbedRawResp struct {
	index int
	Text  string                 `json:"text"`
	Error error                  `json:"error,omitempty"`
	Raw   *api.EmbeddingResponse `json:"raw,omitempty"`
}

// Ollama creates a new Ollama client with the given context and options.
// It initializes the client, validates models, and sets up default models.
// Parameters:
//   - ctx: The context for the client initialization.
//   - opts: Functional options to configure the Ollama client.
//
// Returns:
//   - *Client: The initialized Ollama client.
//   - error: An error if client creation fails.
func Ollama(ctx context.Context, opts ...Option) (*Client, error) {
	b := &builder{Models: make(map[string]llm.Model)}
	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	if len(b.Models) == 0 {
		return nil, ErrNoDefaultModel
	}

	if b.URL == nil {
		return nil, ErrNoBaseURL
	}

	cli := api.NewClient(b.URL, utils.IfElse(
		b.Client == nil, http.DefaultClient, b.Client))

	if err := healthCheck(ctx, cli); err != nil {
		return nil, err
	}

	if b.DefaultEmbed == "" {
		return nil, fmt.Errorf("%w for embedding model", ErrNoDefaultModel)
	}

	if b.DefaultGen == "" { // check if DefaultGen model is set
		return nil, fmt.Errorf("%w for generate model", ErrNoDefaultModel)
	}

	// Validate that the models exist on the Ollama server
	// and also retrieve model capabilities for request validation
	for name, model := range b.Models {
		m, err := cli.Show(ctx, &api.ShowRequest{Model: name})
		if err != nil {
			return nil, fmt.Errorf("%w: %s, %s", ErrModelNotFount, name, err)
		}

		capabilities := make([]string, len(m.Capabilities))
		for i, cap := range m.Capabilities {
			capabilities[i] = string(cap)
		}

		switch model.Type() {
		case llm.ModelEmbed:
			if !slices.Contains(capabilities, "embedding") {
				return nil, fmt.Errorf(
					"%w does not support embedding content: %s", ErrModelNotSupport, name)
			}
		case llm.ModelGenerate:
			if !slices.Contains(capabilities, "completion") {
				return nil, fmt.Errorf(
					"%w generating content: %s", ErrModelNotSupport, name)
			}
		}

		oModel := model.(OllamaModel)
		oModel.License = m.License
		oModel.ModelInfo = m.ModelInfo
		oModel.Capabilities = capabilities
		b.Models[name] = oModel
	}

	base := llm.NewClient()
	for _, model := range b.Models {
		if err := base.WithModel(model); err != nil {
			return nil, err
		}
	}

	if err := base.SetDefaultModel(llm.ModelEmbed, b.DefaultEmbed); err != nil {
		return nil, err
	}

	if err := base.SetDefaultModel(llm.ModelGenerate, b.DefaultGen); err != nil {
		return nil, err
	}
	return &Client{BaseClient: base, OllamaAPI: cli}, nil
}

// Generate produces a response from the Ollama model.
// Parameters:
//   - ctx: The context for the request.
//   - req: llm.GenerateRequest containing the messages and model information.
//
// Returns:
//   - *llm.GenerateResponse with the generated output and raw response.
//   - error if the request fails or the configuration type is invalid.
func (c *Client) Generate(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if len(req.Messages) == 0 {
		return nil, llm.ErrNoInput
	}

	modelName := req.ModelName
	if modelName == "" {
		if m, ok := c.DefaultModel(llm.ModelGenerate); ok {
			modelName = m.Name()
		} else {
			return nil, fmt.Errorf("%w: %s", ErrNoDefaultModel, "generate")
		}
	}

	messages := toOllamaMessages(req.Messages)

	opts, err := toOptions(req.Config)
	if err != nil {
		return nil, err
	}

	if req.Schema != nil {
		if len(opts) == 0 {
			opts = map[string]any{}
		}
		opts["schema"] = req.Schema.S
	}

	isStreaming := false
	var apiResp api.ChatResponse
	if err := c.OllamaAPI.Chat(ctx, &api.ChatRequest{
		Model:    modelName,
		Messages: messages,
		Options:  opts,
		Stream:   &isStreaming,
	}, func(resp api.ChatResponse) error {
		apiResp = resp
		return nil
	}); err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	if !apiResp.Done {
		return nil, ErrIncompleteResponse
	}

	if req.Schema != nil {
		output, err := extractJSONObject(apiResp.Message.Content)
		if err == nil {
			return &llm.GenerateResponse{
				Outputs: []string{output},
				Raw:     apiResp,
			}, nil
		}
	}

	return &llm.GenerateResponse{
		Outputs: []string{apiResp.Message.Content},
		Raw:     apiResp,
	}, nil
}

// Embed generates embeddings for the given request using the Ollama model.
// Parameters:
//   - ctx: The context for the request.
//   - req: llm.EmbedRequest containing the inputs and model information.
//
// Returns:
//   - *llm.EmbedResponse with the generated embeddings and raw response.
//   - error if the request fails or the configuration type is invalid.
func (c *Client) Embed(ctx context.Context, req *llm.EmbedRequest) (*llm.EmbedResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if len(req.Inputs) == 0 {
		return nil, llm.ErrNoInput
	}

	modelName := req.ModelName
	if modelName == "" {
		if m, ok := c.DefaultModel(llm.ModelEmbed); ok {
			modelName = m.Name()
		} else {
			return nil, fmt.Errorf("%w: %s", ErrNoDefaultModel, "embedding")
		}
	}

	opts, err := toOptions(req.Config)
	if err != nil {
		return nil, err
	}

	resp := &llm.EmbedResponse{
		Embeddings: make([]llm.Embedding, len(req.Inputs)),
		Model:      modelName,
	}

	raws := make([]OllamaEmbedRawResp, len(req.Inputs))
	reqCh, respCh := make(chan *OllamaEmbedReq), make(chan *OllamaEmbedRawResp)

	workersWg := sync.WaitGroup{}
	for i := 0; i < Parallel; i++ {
		workersWg.Add(1)
		go func(i int, reqCh <-chan *OllamaEmbedReq, respCh chan<- *OllamaEmbedRawResp) {
			// decrement counter when the goroutine exits.
			defer workersWg.Done()
			for input := range reqCh {
				apiResp, err := c.OllamaAPI.Embeddings(ctx, input.Req)
				respCh <- &OllamaEmbedRawResp{
					index: input.Index,
					Text:  input.Req.Prompt,
					Error: err,
					Raw:   apiResp,
				}
			}
		}(i, reqCh, respCh)
	}

	collectorWg := sync.WaitGroup{}
	collectorWg.Add(1)
	go func(respCh <-chan *OllamaEmbedRawResp) {
		defer collectorWg.Done()
		for rawResp := range respCh {
			if rawResp.Error != nil {
				resp.Embeddings[rawResp.index].State = llm.EmbedStateError
			} else {
				resp.Embeddings[rawResp.index] = llm.Embedding{
					State:  llm.EmbedStateOk,
					Values: utils.ToFloat32(rawResp.Raw.Embedding),
				}
			}
			raws[rawResp.index] = *rawResp
		}
	}(respCh)

	for i, input := range req.Inputs {
		reqCh <- &OllamaEmbedReq{
			Index: i,
			Req: &api.EmbeddingRequest{
				Model:   modelName,
				Prompt:  input.String(),
				Options: opts,
			},
		}
	}
	close(reqCh)
	workersWg.Wait()

	close(respCh)
	collectorWg.Wait()

	resp.Raw = raws
	return resp, nil
}

// BatchGenerate is not supported by Ollama.
func (c *Client) BatchCreate(ctx context.Context, req *llm.BatchRequest) (*llm.BatchResponse, error) {
	return nil, llm.ErrNotImplemented
}

// BatchRetrieve is not supported by Ollama.
func (c *Client) BatchRetrieve(ctx context.Context, req *llm.BatchRetrieveRequest) (*llm.BatchResponse, error) {
	return nil, llm.ErrNotImplemented
}

// BatchCancel is not supported by Ollama.
func (c *Client) BatchCancel(ctx context.Context, req *llm.BatchCancelRequest) error {
	return llm.ErrNotImplemented
}
