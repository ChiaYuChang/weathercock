package ollama

import (
	"fmt"
	"net/http"
	"net/url"
)

type Option func(*builder) error

// WithHost sets the host address for the Ollama server (e.g., "http://localhost:11434").
func WithHost(host string) Option {
	return func(b *builder) error {
		if u, err := url.Parse(host); err != nil {
			return fmt.Errorf("invalid host URL: %w", err)
		} else {
			b.URL = u
		}
		return nil
	}
}

// WithHTTPClient sets a custom http.Client for the Ollama client.
func WithHTTPClient(c *http.Client) Option {
	return func(b *builder) error {
		b.Client = c
		return nil
	}
}

// WithModel registers one or more models with the client.
func WithModel(models ...OllamaModel) Option {
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
