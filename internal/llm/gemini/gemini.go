package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"google.golang.org/genai"
)

const (
	DefaultGenModel   = "gemini-2.5-flash"
	DefaultEmbedModel = "gemini-embedding-001"

	EmbedTaskRetrivalQuery    = "RETRIEVAL_QUERY"
	EmbedTaskRetrivalDocument = "RETRIEVAL_DOCUMENT"
	EmbedTaskClassification   = "CLASSIFICATION"
	EmbedTaskClustering       = "CLUSTERING"
)

const (
	GeminiAPIVersion = "v1beta"
)

var (
	ErrAPIKeyMissing = errors.New("missing Gemini API key")
	ErrModelNotFound = errors.New("model not found")
)

type Client struct {
	*llm.BaseClient
	GenAI *genai.Client
}

type builder struct {
	APIKey       string
	APIVer       string
	Timeout      *time.Duration
	Models       map[string]llm.Model
	DefaultGen   string
	DefaultEmbed string
}

// NewGeminiModel creates a new GeminiModel with the specified model type and name.
func NewGeminiModel(modelType llm.ModelType, name string) GeminiModel {
	return GeminiModel{
		BaseModel: llm.NewBaseModel(modelType, name),
	}
}

// Gemini creates a new Gemini client with the given context and options.
// It initializes the client, validates models, and sets up default models.
// Parameters:
//   - ctx: The context for the client initialization.
//   - opts: Functional options to configure the Gemini client.
//
// Returns:
//   - *Client: The initialized Gemini client.
//   - error: An error if client creation fails.
func Gemini(ctx context.Context, opts ...Option) (*Client, error) {
	b := &builder{}
	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	if b.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	ver := utils.DefaultIfZero(b.APIVer, GeminiAPIVersion)
	cli, err := genai.NewClient(
		ctx,
		&genai.ClientConfig{
			APIKey:  b.APIKey,
			Backend: genai.BackendGeminiAPI,
			HTTPOptions: genai.HTTPOptions{
				APIVersion: ver,
				Timeout:    b.Timeout,
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("could not create Gemini API client: %w", err)
	}

	if len(b.Models) == 0 {
		b.Models = map[string]llm.Model{}
		b.Models[DefaultGenModel] = NewGeminiModel(llm.ModelGenerate, DefaultGenModel)
		b.Models[DefaultEmbedModel] = NewGeminiModel(llm.ModelEmbed, DefaultEmbedModel)
	}

	// validate models
	for name, model := range b.Models {
		m, err := cli.Models.Get(ctx, name, nil)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve model %s from Gemini API: %w", name, err)
		}

		switch model.Type() {
		case llm.ModelEmbed:
			if !slices.Contains(m.SupportedActions, "embedContent") {
				return nil, fmt.Errorf(
					"model %s (%s) does not support embedding content",
					name, m.DisplayName)
			}
		case llm.ModelGenerate:
			if !slices.Contains(m.SupportedActions, "generateContent") {
				return nil, fmt.Errorf(
					"model %s (%s) does not support generating content",
					name, m.DisplayName)
			}
		}

		gModel := model.(GeminiModel)
		gModel.DesplayName = m.DisplayName
		gModel.Version = m.Version
		gModel.Description = m.Description
		gModel.InputTokenLimit = m.InputTokenLimit
		gModel.OutputTokenLimit = m.OutputTokenLimit
		gModel.SupportedActions = m.SupportedActions
		b.Models[name] = gModel
	}

	base := llm.NewClient()
	for _, model := range b.Models {
		if err := base.WithModel(model); err != nil {
			return nil, fmt.Errorf("could not register model %s: %w", model.Name(), err)
		}
	}

	b.DefaultGen = utils.DefaultIfZero(b.DefaultGen, DefaultGenModel)
	if err := base.SetDefaultModel(llm.ModelGenerate, b.DefaultGen); err != nil {
		return nil, fmt.Errorf("could not set default generate model: %w", err)
	}

	b.DefaultEmbed = utils.DefaultIfZero(b.DefaultEmbed, DefaultEmbedModel)
	if err := base.SetDefaultModel(llm.ModelEmbed, b.DefaultEmbed); err != nil {
		return nil, fmt.Errorf("could not set default embed model: %w", err)
	}

	return &Client{base, cli}, nil
}

// Generate sends a content generation request to the Gemini API using the specified model and configuration.
// Parameters:
//   - ctx: The context for the request.
//   - req: llm.GenerateRequest containing the messages and model information.
//
// Returns:
//   - *llm.GenerateResponse with the generated output and raw response.
//   - error if the request fails or the configuration type is invalid.
func (cli *Client) Generate(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if len(req.Messages) == 0 {
		return nil, llm.ErrNoInput
	}

	modelName := req.ModelName
	if modelName == "" {
		if m, ok := cli.DefaultModel(llm.ModelGenerate); ok {
			modelName = m.Name()
		} else {
			modelName = DefaultGenModel
		}
	}

	contents, err := toGenAIContents(req.Messages)
	if err != nil {
		return nil, err
	}

	config, err := assertAs[*genai.GenerateContentConfig](req.Config)
	if err != nil {
		return nil, err
	}

	if req.Schema != nil {
		if config == nil {
			config = &genai.GenerateContentConfig{}
		}
		config.ResponseJsonSchema = req.Schema.S
	}

	resp, err := cli.GenAI.Models.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		return nil, err
	}

	if req.Schema != nil {
		output, err := extractJSONObject(resp.Text())
		if err == nil {
			return &llm.GenerateResponse{
				Outputs: []string{output},
				Raw:     resp,
			}, nil
		}
	}

	return &llm.GenerateResponse{
		Outputs: []string{resp.Text()},
		Raw:     resp,
	}, nil
}

// Embed generates embeddings for the given request using the Gemini API.
// Parameters:
//   - ctx: The context for the request.
//   - req: llm.EmbedRequest containing the inputs and model information.
//
// Returns:
//   - *llm.EmbedResponse with the generated embeddings and raw response.
//   - error if the request fails or the configuration type is invalid.
func (cli *Client) Embed(ctx context.Context, req *llm.EmbedRequest) (*llm.EmbedResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if len(req.Inputs) == 0 {
		return nil, llm.ErrNoInput
	}

	modelName := req.ModelName
	if modelName == "" {
		if m, ok := cli.DefaultModel(llm.ModelEmbed); ok {
			modelName = m.Name()
		} else {
			modelName = DefaultEmbedModel
		}
	}

	contents := make([]*genai.Content, len(req.Inputs))
	for i, input := range req.Inputs {
		contents[i] = genai.NewContentFromText(input.String(), genai.RoleUser)
	}

	config, err := assertAs[*genai.EmbedContentConfig](req.Config)
	if err != nil {
		return nil, err
	}

	resp, err := cli.GenAI.Models.EmbedContent(ctx, modelName, contents, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	embeds := make([]llm.Embedding, len(resp.Embeddings))
	for i, embed := range resp.Embeddings {
		embeds[i] = llm.Embedding{
			State:  llm.EmbedStateOk,
			Values: embed.Values,
		}

		if embed.Statistics != nil && embed.Statistics.Truncated {
			embeds[i].State = llm.EmbedStateTruncated
		}
	}
	return &llm.EmbedResponse{
		Embeddings: embeds,
		Model:      modelName,
		Raw:        resp,
	}, nil
}

// BatchGenerate processes multiple generation requests in a single batch job using the Gemini API.
// Parameters:
//   - ctx: The context for the request.
//   - req: llm.BatchRequest containing multiple generation requests.
//
// Returns:
//   - *llm.BatchResponse with the batch job details.
//   - error if the request fails.
func (cli *Client) BatchCreate(ctx context.Context, req *llm.BatchRequest) (*llm.BatchResponse, error) {
	inlineReqs := make([]*genai.InlinedRequest, len(req.Requests))
	for i, r := range req.Requests {
		switch subreq := r.(type) {
		case *llm.GenerateRequest:
			contents, err := toGenAIContents(subreq.Messages)
			if err != nil {
				return nil, err
			}

			gConf, err := assertAs[*genai.GenerateContentConfig](subreq.Config)
			if err != nil {
				return nil, err
			}

			modelName := subreq.ModelName
			if modelName == "" {
				if m, ok := cli.DefaultModel(llm.ModelGenerate); ok {
					modelName = m.Name()
				} else {
					modelName = DefaultGenModel
				}
			}

			inlineReqs[i] = &genai.InlinedRequest{
				Model:    modelName,
				Contents: contents,
				Config:   gConf,
			}
		case *llm.EmbedRequest:
			return nil, llm.ErrNotImplemented
		default:
			return nil, llm.ErrNotImplemented
		}
	}

	var config *genai.CreateBatchJobConfig
	if req.BatchCreateConfig != nil {
		var err error
		config, err = assertAs[*genai.CreateBatchJobConfig](req.BatchCreateConfig)
		if err != nil {
			return nil, err
		}
		config.DisplayName = req.BatchJobName
	} else {
		config = &genai.CreateBatchJobConfig{
			DisplayName: req.BatchJobName,
		}
	}

	modelName := req.ModelName
	if modelName == "" {
		if m, ok := cli.DefaultModel(llm.ModelGenerate); ok {
			modelName = m.Name()
		} else {
			modelName = DefaultGenModel
		}
	}

	src := &genai.BatchJobSource{
		// Format:          "jsonl",
		InlinedRequests: inlineReqs,
	}

	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	_, err = req.ReadWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write to writer: %w", err)
	}

	job, err := cli.GenAI.Batches.Create(ctx, modelName, src, config)
	if err != nil {
		return nil, err
	}

	return &llm.BatchResponse{
		ID:        job.Name,
		Status:    string(job.State),
		IsDone:    IsTerminalJobState(job.State),
		CreatedAt: job.CreateTime,
		StartAt:   job.StartTime,
		EndAt:     job.EndTime,
		UpdateAt:  job.UpdateTime,
		Responses: nil,
		Raw:       job,
	}, err
}

// BatchRetrieve retrieves the status and results of a previously submitted batch job from the Gemini API.
// Parameters:
//   - ctx: The context for the request.
//   - req: llm.BatchRetrieveRequest containing the ID of the batch job to retrieve.
//
// Returns:
//   - *llm.BatchResponse with the batch job details and results if completed.
//   - error if the request fails.
func (cli *Client) BatchRetrieve(ctx context.Context, req *llm.BatchRetrieveRequest) (*llm.BatchResponse, error) {
	conf, err := assertAs[*genai.GetBatchJobConfig](req.RetrieveConfig)
	if err != nil {
		return nil, err
	}

	job, err := cli.GenAI.Batches.Get(ctx, req.ID, conf)
	if err != nil {
		return nil, err
	}

	var responses [][]byte
	if job.State == genai.JobStateSucceeded {
		for _, resp := range job.Dest.InlinedResponses {
			responses = append(responses, []byte(resp.Response.Text()))
		}
	}

	return &llm.BatchResponse{
		ID:        job.Name,
		Status:    string(job.State),
		IsDone:    IsTerminalJobState(job.State),
		CreatedAt: job.CreateTime,
		StartAt:   job.StartTime,
		EndAt:     job.EndTime,
		UpdateAt:  job.UpdateTime,
		Responses: responses,
		Raw:       job,
	}, err
}

// BatchCancel cancels a running batch job on the Gemini API.
// Parameters:
//   - ctx: The context for the request.
//   - req: llm.BatchCancelRequest containing the ID of the batch job to cancel.
//
// Returns:
//   - error if the request fails.
func (cli *Client) BatchCancel(ctx context.Context, req *llm.BatchCancelRequest) error {
	config, err := assertAs[*genai.CancelBatchJobConfig](req.Config)
	if err != nil {
		return err
	}
	return cli.GenAI.Batches.Cancel(ctx, req.ID, config)
}
