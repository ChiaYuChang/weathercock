package llm

import (
	"context"
	"encoding/json"
)

type LLM struct {
	ChatCompletion
	Embedding
}

type ChatCompletion interface {
	New(ctx context.Context, model string, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
}

type ResponseRequest struct {
	Model           string            `json:"model"`
	Input           string            `json:"input"`                       // input text for the model
	Instruction     string            `json:"instruction,omitempty"`       // optional instruction to guide the model's response
	MaxOutputTokens int               `json:"max_output_tokens,omitempty"` // maximum number of tokens to generate in the response
	Metadata        map[string]string `json:"metadata,omitempty"`          // additional metadata for the request, can be used for tracking or logging
	Options         map[string]any    `json:"options,omitempty"`           // additional options for the request, can include parameters like temperature, top_p, etc.
	Temperature     float64           `json:"temperature,omitempty"`       // temperature for sampling, higher values mean more randomness
	Schema          any               `json:"schema,omitempty"`            // JSON schema for the response, can be used to validate the response structure

}

type ModelOptions struct {
	Temperature     float64 `json:"temperature,omitempty"`       // temperature for sampling, higher values mean more randomness
	MaxOutputTokens int     `json:"max_output_tokens,omitempty"` // maximum number of tokens to generate in the response
	TopLogProbs     int     `json:"top_logprobs,omitempty"`      // number of top log probabilities to return, useful for debugging or analysis
	TopP            float64 `json:"top_p,omitempty"`             // top-p sampling,
	Seed            int32   `json:"seed,omitempty"`              // seed for random number generation, useful for reproducibility
}

type Reasoning struct {
	// effort level of the reasoning process, should be one of "low",
	// "medium", "high"
	Effort string `json:"effort,omitempty"`
	// summary of the reasoning process, should be one of "auto",
	// "concise", "detailed"
	Summary string `json:"summary,omitempty"`
}

type ChatCompletionRequest struct {
	Model             string                  `json:"model"`                        // model to use for the chat completion
	SystemInstruction []string                `json:"system_instruction,omitempty"` // system message to set the context for the chat
	Messages          []ChatCompletionMessage `json:"messages"`                     // list of messages in the chat, each message should implement ChatCompletionMessage interface
	Stream            bool                    `json:"stream,omitempty"`             // whether to stream the response or return it all at once
	Schema            any                     `json:"schema,omitempty"`             // JSON schema for the response, can be used to validate the response structure
	ModelOptions
}

type ChatCompletionMessage interface {
	json.Marshaler
	Role() string
	Content() []string
}

type ChatCompletionResponse struct {
	ID       string // identifier for the chat completion
	Model    string // model used for the chat completion
	Messages []ChatCompletionMessage
	Thinking string
	resp     any // raw response from the LLM, can be used for debugging or further processing
}

func (resp *ChatCompletionResponse) OriginalResponse() any {
	if resp == nil {
		return nil
	}
	return resp.resp
}

type Embedding interface {
	New(ctx context.Context, model string, req *EmbeddingRequest) (*EmbeddingResponse, error)
}

type EmbeddingRequest struct {
	Model string
	Input []string
}

type EmbeddingResponse struct {
	ID         string
	Model      string
	Embeddings [][]float64
}
