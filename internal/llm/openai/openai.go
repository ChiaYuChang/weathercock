package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared"
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
	ErrFailedToGetOutputFile = errors.New("failed to get output file")
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

func (cli *Client) Generate(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if len(req.Messages) == 0 {
		return nil, llm.ErrNoInput
	}

	if cli.UseChatComplete {
		return cli.generateChatCompletions(ctx, req)
	}
	return cli.generateRequest(ctx, req)
}

// generateRequest produces a response from an OpenAI model.
func (cli *Client) generateRequest(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
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

	resp, err := cli.OpenAI.Responses.New(
		ctx,
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
		Outputs: []string{resp.OutputText()},
		Raw:     resp,
	}, nil
}

func (cli *Client) generateChatCompletions(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
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

	resp, err := cli.OpenAI.Chat.Completions.New(
		ctx,
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
		Outputs: []string{resp.Choices[0].Message.Content},
		Raw:     resp,
	}, nil
}

// Embed generates embeddings for the given request using an OpenAI model.
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
	if cli.EmbedDim > 0 {
		resp, err = cli.OpenAI.Embeddings.New(
			ctx,
			openai.EmbeddingNewParams{
				Input: openai.EmbeddingNewParamsInputUnion{
					OfArrayOfStrings: input,
				},
				Model:          modelName,
				Dimensions:     openai.Int(int64(cli.EmbedDim)),
				EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
			},
			opts...,
		)
	} else {
		resp, err = cli.OpenAI.Embeddings.New(
			ctx,
			openai.EmbeddingNewParams{
				Input: openai.EmbeddingNewParamsInputUnion{
					OfArrayOfStrings: input,
				},
				Model:          modelName,
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

type BatchRequestJSONL struct {
	CustomID string                        `json:"custom_id"`
	Method   string                        `json:"method"`
	Endpoint openai.BatchNewParamsEndpoint `json:"url"`
	Body     any                           `json:"body"`
}

func (jsonl BatchRequestJSONL) ContentType() string {
	return "application/jsonl"
}

func (cli *Client) File(ctx context.Context, r io.Reader, filename, contenttype string, opts ...option.RequestOption) (res *openai.FileObject, err error) {
	file, err := cli.OpenAI.Files.New(
		ctx,
		openai.FileNewParams{
			File:    openai.File(r, filename, contenttype),
			Purpose: openai.FilePurposeBatch,
		},
		opts...,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create %s (content type: %s) for batch: %w", filename, contenttype, err)
	}
	return file, nil
}

func (cli *Client) BatchCreate(ctx context.Context, req *llm.BatchRequest) (*llm.BatchResponse, error) {
	if req == nil {
		return nil, llm.ErrRequestShouldNotBeNull
	}

	if len(req.Requests) == 0 {
		return nil, llm.ErrNoInput
	}

	if req.ReadWriter == nil {
		return nil, fmt.Errorf("read writer should not be nil")
	}

	modelName := req.ModelName
	if modelName == "" {
		if m, ok := cli.DefaultModel(llm.ModelEmbed); ok {
			modelName = m.Name()
		} else {
			modelName = DefaultGenModel
		}
	}

	formatter := "%d-%s-" + formatter(len(req.Requests))
	now := time.Now().Unix()
	for i, r := range req.Requests {
		var body any
		var jsonl BatchRequestJSONL

		switch subr := r.(type) {
		case *llm.GenerateRequest:
			body = responses.ResponseNewParams{
				Model: modelName,
				Input: responses.ResponseNewParamsInputUnion{
					OfInputItemList: toResponseInputParam(subr.Messages),
				},
			}

			jsonl = BatchRequestJSONL{
				CustomID: fmt.Sprintf("gen-"+formatter, now, req.BatchJobName, i),
				Endpoint: openai.BatchNewParamsEndpointV1Responses,
			}
		case *llm.EmbedRequest:
			input := make([]string, len(subr.Inputs))
			for i := range subr.Inputs {
				input[i] = subr.Inputs[i].String()
			}

			tmp := openai.EmbeddingNewParams{
				Input: openai.EmbeddingNewParamsInputUnion{
					OfArrayOfStrings: input,
				},
				Model:          modelName,
				EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
			}
			if cli.EmbedDim > 0 {
				tmp.Dimensions = openai.Int(cli.EmbedDim)
			}
			body = tmp
			jsonl = BatchRequestJSONL{
				CustomID: fmt.Sprintf("embed-"+formatter, now, req.BatchJobName, i),
				Endpoint: openai.BatchNewParamsEndpointV1Embeddings,
			}
		default:
			return nil, fmt.Errorf("%s: %T", llm.ErrNotImplemented, r)
		}

		jsonl.Method = http.MethodPost
		jsonl.Body = body

		data, err := json.Marshal(jsonl)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal %d-th request to jsonl: %w", i, err)
		}

		req.ReadWriter.Write(data)
		req.ReadWriter.Write([]byte{'\n'})
	}

	var opts []option.RequestOption
	if v, ok := req.BatchCreateConfig.([]option.RequestOption); ok {
		opts = v
	}

	if f, ok := req.ReadWriter.(*os.File); ok {
		f.Seek(0, io.SeekStart)
	}

	file, err := cli.File(ctx, req.ReadWriter, fmt.Sprintf("%d-%s.jsonl", now, req.BatchJobName), "application/jsonl", opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create jsonl file for batch: %w", err)
	}

	metadata := shared.Metadata{
		"name": req.BatchJobName,
	}
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	batch, err := cli.OpenAI.Batches.New(
		ctx,
		openai.BatchNewParams{
			CompletionWindow: openai.BatchNewParamsCompletionWindow24h,
			Endpoint:         openai.BatchNewParamsEndpoint(req.Endpoint),
			InputFileID:      file.ID,
			Metadata:         metadata,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	return &llm.BatchResponse{
		ID:           batch.ID,
		OutputFileID: batch.OutputFileID,
		InputFileID:  batch.InputFileID,
		Status:       string(batch.Status),
		IsDone:       IsTerminalJobState(batch.Status),
		CreatedAt:    time.Unix(batch.CreatedAt, 0),
		StartAt:      time.Unix(batch.InProgressAt, 0),
		EndAt:        time.Unix(batch.CompletedAt, 0),
		UpdateAt:     time.Unix(batch.CreatedAt, 0),
		Responses:    nil,
		Raw: map[string]any{
			"file":  file,
			"batch": batch,
		},
	}, nil
}

func (cli *Client) BatchRetrieve(ctx context.Context, req *llm.BatchRetrieveRequest) (*llm.BatchResponse, error) {
	var opts []option.RequestOption
	if v, ok := req.StatusCheckConfig.([]option.RequestOption); ok {
		opts = v
	}

	batch, err := cli.OpenAI.Batches.Get(ctx, req.ID, opts...)
	if err != nil {
		if e, ok := err.(*openai.Error); ok {
			return &llm.BatchResponse{
				HTTPStatusCode: e.StatusCode,
				HTTPMessage:    e.Message,
				ID:             req.ID,
			}, e
		}
		return nil, fmt.Errorf("failed to retrive batch: %w", err)
	}

	resp := &llm.BatchResponse{
		ID:           batch.ID,
		InputFileID:  batch.InputFileID,
		OutputFileID: batch.OutputFileID,
		Status:       string(batch.Status),
		IsDone:       IsTerminalJobState(batch.Status),
		CreatedAt:    time.Unix(batch.CreatedAt, 0),
		StartAt:      time.Unix(batch.InProgressAt, 0),
		EndAt:        time.Unix(batch.CompletedAt, 0),
		UpdateAt:     time.Unix(batch.CreatedAt, 0),
		Responses:    nil,
		Raw: map[string]any{
			"batch": batch,
		},
	}

	if !IsTerminalJobState(batch.Status) {
		return resp, nil
	}

	if resp.Status != string(openai.BatchStatusCompleted) {
		batch.OutputFileID = batch.ErrorFileID
	}

	if batch.OutputFileID == "" {
		return resp, fmt.Errorf("%w: empty file_id field", ErrFailedToGetOutputFile)
	}

	opts = nil
	if v, ok := req.StatusCheckConfig.([]option.RequestOption); ok {
		opts = v
	}

	file, err := cli.OpenAI.Files.Content(ctx, batch.OutputFileID, opts...)
	resp.Raw = file
	if err != nil {
		return resp, fmt.Errorf("%s: %w", ErrFailedToGetOutputFile.Error(), err)
	}

	if file.StatusCode != http.StatusOK {
		defer file.Body.Close()
		body, err := io.ReadAll(file.Body)
		if err != nil {
			return resp, fmt.Errorf("failed to get output file: %s (%d), body: read error: %w", file.Status, file.StatusCode, err)
		}
		return resp, fmt.Errorf("failed to get output file: %s (%d), body: %s", file.Status, file.StatusCode, string(body))
	}

	defer file.Body.Close()
	body, err := io.ReadAll(file.Body)
	if err != nil {
		return resp, fmt.Errorf("failed to get output file: %w", err)
	}
	resp.Responses = bytes.Split(body, []byte("\n"))
	return resp, nil
}

func (cli *Client) BatchCancel(ctx context.Context, req *llm.BatchCancelRequest) error {
	_, err := cli.OpenAI.Batches.Cancel(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("failed to cancel batch %s: %w", req.ID, err)
	}
	return nil
}

func formatter(n int) string {
	digit := 0
	for ; n > 0; n /= 10 {
		digit++
	}
	return fmt.Sprintf("%%0%dd", digit)
}

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

// IsTerminalJobState checks if a given job status indicates a terminal state (succeeded, failed, cancelled, or expired).
func IsTerminalJobState(status openai.BatchStatus) bool {
	switch status {
	case openai.BatchStatusCompleted,
		openai.BatchStatusFailed,
		openai.BatchStatusCancelled,
		openai.BatchStatusExpired:
		return true
	default:
		return false
	}
}
