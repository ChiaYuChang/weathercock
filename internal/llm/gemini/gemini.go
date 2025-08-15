package gemini

import (
	"context"
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

func NewGeminiModel(modelType llm.ModelType, name string) GeminiModel {
	return GeminiModel{
		BaseModel: llm.NewBaseModel(modelType, name),
	}
}

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
//   - req: llm.GenerateRequest containing the messages and model information.
//
// Returns:
//   - *llm.GenerateResponse with the generated output and raw response.
//   - error if the request fails or the configuration type is invalid.
func (cli *Client) Generate(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
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

	gConf, err := assertAs[*genai.GenerateContentConfig](req.Config)
	if err != nil {
		return nil, err
	}

	gResp, err := cli.GenAI.Models.GenerateContent(ctx, modelName, contents, gConf)
	if err != nil {
		return nil, err
	}

	return &llm.GenerateResponse{
		Outputs: []string{gResp.Text()},
		Raw:     gResp,
	}, nil
}

func (cli *Client) Embed(ctx context.Context, req *llm.EmbedRequest) (*llm.EmbedResponse, error) {
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

	gConf, err := assertAs[*genai.EmbedContentConfig](req.Config)
	if err != nil {
		return nil, err
	}

	gResp, err := cli.GenAI.Models.EmbedContent(ctx, modelName, contents, gConf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	resp := &llm.EmbedResponse{
		Embeddings: make([]llm.Embedding, len(gResp.Embeddings)),
		Model:      modelName,
		Raw:        gResp,
	}

	for i, embed := range gResp.Embeddings {
		resp.Embeddings[i] = llm.Embedding{
			State:  llm.EmbedStateOk,
			Values: embed.Values,
		}

		if embed.Statistics != nil && embed.Statistics.Truncated {
			resp.Embeddings[i].State = llm.EmbedStateTruncated
		}
	}
	return resp, nil
}

func (cli *Client) BatchGenerate(ctx context.Context, req *llm.BatchRequest) (*llm.BatchResponse, error) {
	inlineReqs := make([]*genai.InlinedRequest, len(req.Requests))
	for i, r := range req.Requests {
		contents, err := toGenAIContents(r.Messages)
		if err != nil {
			return nil, err
		}

		gConf, err := assertAs[*genai.GenerateContentConfig](r.Config)
		if err != nil {
			return nil, err
		}

		modelName := r.ModelName
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
	}

	var gConf *genai.CreateBatchJobConfig
	if req.Config != nil {
		gConf, err := assertAs[*genai.CreateBatchJobConfig](req.Config)
		if err != nil {
			return nil, err
		}
		gConf.DisplayName = req.BatchJobName
	} else {
		gConf = &genai.CreateBatchJobConfig{
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
	job, err := cli.GenAI.Batches.Create(
		ctx, modelName, &genai.BatchJobSource{InlinedRequests: inlineReqs}, gConf,
	)
	if err != nil {
		return nil, err
	}

	return &llm.BatchResponse{
		ID:        job.Name,
		Status:    string(job.State),
		IsDone:    job.State == genai.JobStateSucceeded,
		CreatedAt: job.CreateTime,
		StartAt:   job.StartTime,
		EndAt:     job.EndTime,
		UpdateAt:  job.UpdateTime,
		Raw:       job,
	}, err
}

func (cli *Client) BatchRetrieve(ctx context.Context, req *llm.BatchRetrieveRequest) (*llm.BatchResponse, error) {
	conf, err := assertAs[*genai.GetBatchJobConfig](req.Config)
	if err != nil {
		return nil, err
	}

	job, err := cli.GenAI.Batches.Get(ctx, req.ID, conf)
	if err != nil {
		return nil, err
	}

	var responses []*llm.GenerateResponse
	if job.State == genai.JobStateSucceeded {
		responses = make([]*llm.GenerateResponse, len(job.Dest.InlinedResponses))
		for i, resp := range job.Dest.InlinedResponses {
			responses[i] = &llm.GenerateResponse{
				Outputs: []string{resp.Response.Text()},
				Raw:     resp,
			}
		}
	}

	return &llm.BatchResponse{
		ID:        job.Name,
		Status:    string(job.State),
		IsDone:    isTerminalJobState(string(job.State)),
		CreatedAt: job.CreateTime,
		StartAt:   job.StartTime,
		EndAt:     job.EndTime,
		UpdateAt:  job.UpdateTime,
		Responses: responses,
		Raw:       job,
	}, err
}

func (cli *Client) BatchCancel(ctx context.Context, req *llm.BatchCancelRequest) error {
	gConf, err := assertAs[*genai.CancelBatchJobConfig](req.Config)
	if err != nil {
		return err
	}
	return cli.GenAI.Batches.Cancel(ctx, req.ID, gConf)
}
