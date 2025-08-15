package llm

import "context"

// LLM defines the interface for Large Language Model operations and model management.
// Implementations should provide methods for text generation, embedding, and model handling.
type LLM interface {
	// Generate produces a response from the LLM given a request.
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// BatchGenerate processes multiple generation requests in a single call.
	BatchGenerate(ctx context.Context, reqs *BatchRequest) (*BatchResponse, error)

	BatchRetrieve(ctx context.Context, req *BatchRetrieveRequest) (*BatchResponse, error)

	BatchCancel(ctx context.Context, req *BatchCancelRequest) error

	// Embed generates embeddings for the given request.
	Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error)

	// AddModel registers a new model with the LLM service.
	AddModel(model Model)

	// SetDefaultModel sets the default model for a given type.
	SetDefaultModel(modelType ModelType, name string) error

	// HasModel checks if a model with the given name exists.
	HasModel(name string) bool

	// DefaultModel returns the default model for a given type, if set.
	DefaultModel(modelType ModelType) (Model, bool)

	// ListModels returns all registered models.
	ListModels() []Model
}
