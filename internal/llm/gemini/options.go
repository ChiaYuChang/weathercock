package gemini

import (
	"fmt"
	"time"
)

var (
	ErrDuplicatedModel = fmt.Errorf("duplicate model")
)

type Option func(*builder) error

// WithAPIKey sets the API key for Gemini authentication.
func WithAPIKey(apikey string) Option {
	return func(b *builder) error {
		b.APIKey = apikey
		return nil
	}
}

// WithModel registers one or more Gemini models with the client.
func WithModel(models ...GeminiModel) Option {
	return func(b *builder) error {
		for _, model := range models {
			if _, exists := b.Models[model.Name()]; exists {
				return fmt.Errorf("%w: %s", ErrDuplicatedModel, model.Name())
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

// WithAPIVersion sets the API version for the Gemini client.
func WithAPIVersion(ver string) Option {
	return func(b *builder) error {
		b.APIVer = ver
		return nil
	}
}

// WithTimeout sets the timeout for API requests.
func WithTimeout(timeout time.Duration) Option {
	return func(b *builder) error {
		b.Timeout = &timeout
		return nil
	}
}
