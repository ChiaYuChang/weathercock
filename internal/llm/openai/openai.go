package openai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/responses"
)

const (
	DefaultGenModel   = openai.ChatModelGPT5Nano
	DefaultEmbedModel = openai.EmbeddingModelTextEmbedding3Small
)

var (
	MaxRetries          = 4
	MaxRetryWaitingTime = 10 * time.Second
)

var (
	ErrAPIKeyMissing         = errors.New("OpenAI API key is required")
	ErrCanNotConnectToServer = errors.New("can not connect to server")
)

// Client implements the llm.LLM interface for OpenAI.
type Client struct {
	*llm.BaseClient
	OpenAI          openai.Client
	EmbedDim        int64
	UseChatComplete bool
}

// builder is used to construct an OpenAI Client using the functional options pattern.
type builder struct {
	APIKey          string
	BaseURL         *url.URL
	HTTPClient      *http.Client
	Models          map[string]llm.Model
	Timeout         time.Duration
	MaxRetries      int
	Header          map[string]string
	Middleware      []option.Middleware
	UseChatComplete bool
	EmbedDim        int64
	DefaultGen      string
	DefaultEmbed    string
}

type OpenAIModel struct {
	llm.BaseModel
}

func NewOpenAIModel(modelType llm.ModelType, name string) OpenAIModel {
	return OpenAIModel{
		BaseModel: llm.NewBaseModel(modelType, name),
	}
}

type Option func(*builder) error

// WithAPIKey sets the API key for OpenAI authentication.
func WithAPIKey(key string) Option {
	return func(b *builder) error {
		b.APIKey = key
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(b *builder) error {
		b.Timeout = timeout
		return nil
	}
}

func WithMaxRetries(retries int) Option {
	return func(b *builder) error {
		if retries <= 0 {
			return fmt.Errorf("max retries must be a positive integer, got %d", retries)
		}
		b.MaxRetries = retries
		return nil
	}
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(c *http.Client) Option {
	return func(b *builder) error {
		b.HTTPClient = c
		return nil
	}
}

// WithModel registers one or more models with the client.
func WithModel(models ...OpenAIModel) Option {
	return func(b *builder) error {
		for _, model := range models {
			if _, exists := b.Models[model.Name()]; exists {
				return fmt.Errorf("duplicate model: %s", model.Name())
			}
			b.Models[model.Name()] = model
		}
		return nil
	}
}

// WithDefaultGenerate sets the default model for text generation.
func WithDefaultGenerate(name string) Option {
	return func(b *builder) error {
		b.DefaultGen = name
		return nil
	}
}

// WithDefaultEmbed sets the default model for embeddings.
func WithDefaultEmbed(name string) Option {
	return func(b *builder) error {
		b.DefaultEmbed = name
		return nil
	}
}

func UseChatChatCompletions() Option {
	return func(b *builder) error {
		b.UseChatComplete = true
		return nil
	}
}

func WithEmbedDim(dim int) Option {
	return func(b *builder) error {
		if dim <= 0 {
			return fmt.Errorf("embedding dimension should be greater then zero: %d", dim)
		}
		b.EmbedDim = int64(dim)
		return nil
	}
}

func WithBaseURL(u string) Option {
	return func(b *builder) error {
		u, err := url.Parse(u)
		if err != nil {
			return err
		}
		b.BaseURL = u
		return nil
	}
}

// OpenAI creates a new OpenAI client.
func OpenAI(ctx context.Context, opts ...Option) (*Client, error) {
	b := &builder{Models: make(map[string]llm.Model)}
	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	if b.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	openAICliOptions := []option.RequestOption{}
	if b.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}
	openAICliOptions = append(openAICliOptions, option.WithAPIKey(b.APIKey))
	if b.Timeout > 0 {
		openAICliOptions = append(openAICliOptions, option.WithRequestTimeout(b.Timeout))
	}

	if b.BaseURL != nil {
		openAICliOptions = append(openAICliOptions, option.WithBaseURL(b.BaseURL.String()))
	}

	if b.HTTPClient != nil {
		openAICliOptions = append(openAICliOptions, option.WithHTTPClient(b.HTTPClient))
	}

	if b.MaxRetries > 0 {
		openAICliOptions = append(openAICliOptions, option.WithMaxRetries(b.MaxRetries))
	}

	if b.Header != nil {
		for k, v := range b.Header {
			openAICliOptions = append(openAICliOptions, option.WithHeader(k, v))
		}
	}

	if b.Middleware != nil {
		openAICliOptions = append(openAICliOptions, option.WithMiddleware(b.Middleware...))
	}
	cli := openai.NewClient(openAICliOptions...)

	if err := healthCheck(ctx, cli); err != nil {
		return nil, err
	}

	// Add default models if none were provided by the user.
	if len(b.Models) == 0 {
		b.Models[DefaultGenModel] = NewOpenAIModel(llm.ModelGenerate, DefaultGenModel)
		b.Models[DefaultEmbedModel] = NewOpenAIModel(llm.ModelEmbed, DefaultEmbedModel)
	}

	base := llm.NewClient()
	for _, model := range b.Models {
		if err := base.WithModel(model); err != nil {
			return nil, err
		}
	}

	b.DefaultGen = utils.DefaultIfZero(b.DefaultGen, DefaultGenModel)
	if err := base.SetDefaultModel(llm.ModelGenerate, b.DefaultGen); err != nil {
		return nil, err
	}

	b.DefaultEmbed = utils.DefaultIfZero(b.DefaultEmbed, DefaultEmbedModel)
	if err := base.SetDefaultModel(llm.ModelEmbed, b.DefaultEmbed); err != nil {
		return nil, err
	}

	return &Client{
		BaseClient:      base,
		OpenAI:          cli,
		EmbedDim:        b.EmbedDim,
		UseChatComplete: b.UseChatComplete,
	}, nil
}

func (cli *Client) Generate(req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if req.Context == nil {
		return nil, llm.ErrContextShouldNotBeNull
	}

	if len(req.Messages) == 0 {
		return nil, llm.ErrNoInput
	}

	if cli.UseChatComplete {
		return cli.generateChatCompletions(req)
	}
	return cli.generateRequest(req)
}

// generateRequest produces a response from an OpenAI model.
func (cli *Client) generateRequest(req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	modelName := req.ModelName
	if modelName == "" {
		if m, ok := cli.DefaultModel(llm.ModelGenerate); ok {
			modelName = m.Name()
		} else {
			modelName = DefaultGenModel
		}
	}

	var opts []option.RequestOption
	if v, ok := req.Config.([]option.RequestOption); ok {
		opts = v
	}

	oResp, err := cli.OpenAI.Responses.New(
		req.Context,
		responses.ResponseNewParams{
			Model: modelName,
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: toResponseInputParam(req.Messages),
			},
		},
		opts...,
	)

	if err != nil {
		if e, ok := err.(*openai.Error); ok {
			return nil, fmt.Errorf("code: %s (%d), type: %s, msg: %s",
				e.Code, e.StatusCode, e.Type, e.Message)
		}
		return nil, err
	}

	return &llm.GenerateResponse{
		Outputs: []string{oResp.OutputText()},
		Raw:     oResp,
	}, nil
}

func (cli *Client) generateChatCompletions(req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	modelName := req.ModelName
	if modelName == "" {
		if m, ok := cli.DefaultModel(llm.ModelGenerate); ok {
			modelName = m.Name()
		} else {
			modelName = DefaultGenModel
		}
	}

	messages := []openai.ChatCompletionMessageParamUnion{}
	for _, msg := range req.Messages {
		for _, content := range msg.Content {
			switch msg.Role {
			case llm.RoleSystem:
				messages = append(messages, openai.SystemMessage(content))
			case llm.RoleAssistant:
				messages = append(messages, openai.AssistantMessage(content))
			case llm.RoleUser:
				messages = append(messages, openai.UserMessage(content))
			}
		}
	}

	var opts []option.RequestOption
	if v, ok := req.Config.([]option.RequestOption); ok {
		opts = v
	}

	oResp, err := cli.OpenAI.Chat.Completions.New(
		req.Context,
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    modelName,
		},
		opts...,
	)

	if err != nil {
		if e, ok := err.(*openai.Error); ok {
			return nil, fmt.Errorf("code: %s (%d), type: %s, msg: %s",
				e.Code, e.StatusCode, e.Type, e.Message)
		}
		return nil, err
	}

	return &llm.GenerateResponse{
		Outputs: []string{oResp.Choices[0].Message.Content},
		Raw:     oResp,
	}, nil
}

// Embed generates embeddings for the given request using an OpenAI model.
func (c *Client) Embed(req *llm.EmbedRequest) (*llm.EmbedResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if req.Ctx == nil {
		return nil, llm.ErrContextShouldNotBeNull
	}

	if len(req.Inputs) == 0 {
		return nil, llm.ErrNoInput
	}

	modelName := req.ModelName
	if modelName == "" {
		if m, ok := c.DefaultModel(llm.ModelEmbed); ok {
			modelName = m.Name()
		} else {
			modelName = DefaultGenModel
		}
	}

	input := make([]string, len(req.Inputs))
	for i := range req.Inputs {
		input[i] = req.Inputs[i].String()
	}

	var opts []option.RequestOption
	if v, ok := req.Config.([]option.RequestOption); ok {
		opts = v
	}

	var resp *openai.CreateEmbeddingResponse
	var err error
	if c.EmbedDim > 0 {
		resp, err = c.OpenAI.Embeddings.New(
			req.Ctx,
			openai.EmbeddingNewParams{
				Input: openai.EmbeddingNewParamsInputUnion{
					OfArrayOfStrings: input,
				},
				Model:          modelName,
				EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
			},
			opts...,
		)
	} else {
		resp, err = c.OpenAI.Embeddings.New(
			req.Ctx,
			openai.EmbeddingNewParams{
				Input: openai.EmbeddingNewParamsInputUnion{
					OfArrayOfStrings: input,
				},
				Model:          modelName,
				Dimensions:     openai.Int(int64(c.EmbedDim)),
				EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
			},
			opts...,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	embedding := make([]llm.Embedding, len(resp.Data))
	for _, d := range resp.Data {
		embedding[d.Index] = llm.Embedding{
			State:  llm.EmbedStateOk,
			Values: utils.ToFloat32(d.Embedding),
		}
	}

	return &llm.EmbedResponse{
		Model:      modelName,
		Embeddings: embedding,
		Raw:        resp,
	}, nil
}

// decodeBase64ToFloat32 decodes a base64 string into a slice of float32.
// OpenAI embeddings are little-endian.
// func decodeBase64ToFloat32(b64 string) ([]float32, error) {
// 	decoded, err := base64.StdEncoding.DecodeString(b64)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode base64 string: %w", err)
// 	}

// 	if len(decoded)%4 != 0 {
// 		return nil, fmt.Errorf("decoded byte slice length is not a multiple of 4, got %d", len(decoded))
// 	}

// 	count := len(decoded) / 4
// 	floats := make([]float32, count)
// 	for i := 0; i < count; i++ {
// 		bits := binary.LittleEndian.Uint32(decoded[i*4 : (i+1)*4])
// 		floats[i] = math.Float32frombits(bits)
// 	}
// 	return floats, nil
// }

// // BatchGenerate is not yet implemented for OpenAI.
// func (c *Client) BatchGenerate(req *llm.BatchRequest) (*llm.BatchResponse, error) {
// 	return nil, errNotImplemented
// }

// // BatchRetrieve is not yet implemented for OpenAI.
// func (c *Client) BatchRetrieve(req *llm.BatchRetrieveRequest) (*llm.BatchResponse, error) {
// 	return nil, errNotImplemented
// }

// // BatchCancel is not yet implemented for OpenAI.
// func (c *Client) BatchCancel(req *llm.BatchCancelRequest) error {
// 	return errNotImplemented
// }

// // toOpenAIMessages converts the internal message format to the OpenAI format.
// func toOpenAIMessages(messages []llm.Message) []openai.ChatCompletionMessage {
// 	apiMessages := make([]openai.ChatCompletionMessage, len(messages))
// 	for i, msg := range messages {
// 		var role string
// 		switch msg.Role {
// 		case llm.RoleSystem:
// 			role = openai.ChatMessageRoleSystem
// 		case llm.RoleUser:
// 			role = openai.ChatMessageRoleUser
// 		case llm.RoleAssistant:
// 			role = openai.ChatMessageRoleAssistant
// 		default:
// 			// Default to user role if role is unspecified or unknown
// 			role = openai.ChatMessageRoleUser
// 		}
// 		apiMessages[i] = openai.ChatCompletionMessage{
// 			Role:    role,
// 			Content: msg.Content[0], // Assuming single content for simplicity
// 		}
// 	}
// 	return apiMessages
// }

func healthCheck(ctx context.Context, cli openai.Client) error {
	var err error
	for i := 0; i < MaxRetries; i++ {
		if _, err = cli.Models.List(ctx); err == nil {
			return nil
		}
		time.Sleep(min(1<<i*time.Second, MaxRetryWaitingTime))
	}
	return ErrCanNotConnectToServer
}

func toResponseInputParam(msgs []llm.Message) responses.ResponseInputParam {
	param := make(responses.ResponseInputParam, len(msgs))
	for i, msg := range msgs {
		content := make(responses.ResponseInputMessageContentListParam, len(msg.Content))
		role := "user"
		if msg.Role == llm.RoleAssistant || msg.Role == llm.RoleSystem {
			role = "system"
		}

		for j, c := range msg.Content {
			content[j] = responses.ResponseInputContentUnionParam{
				OfInputText: &responses.ResponseInputTextParam{
					Text: c,
				},
			}
		}

		param[i] = responses.ResponseInputItemUnionParam{
			OfInputMessage: &responses.ResponseInputItemMessageParam{
				Role:    role,
				Content: content,
			},
		}
	}
	return param
}
