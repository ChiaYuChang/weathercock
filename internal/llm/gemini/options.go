package gemini

import (
	"fmt"
	"time"
)

var (
	ErrDuplicatedModel = fmt.Errorf("duplicate model")
)

type Option func(*builder) error

func WithAPIKey(apikey string) Option {
	return func(b *builder) error {
		b.APIKey = apikey
		return nil
	}
}

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

func WithDefaultGenerate(name string) Option {
	return func(b *builder) error {
		b.DefaultGen = name
		return nil
	}
}

func WithDefaultEmbed(name string) Option {
	return func(b *builder) error {
		b.DefaultEmbed = name
		return nil
	}
}

func WithAPIVersion(ver string) Option {
	return func(b *builder) error {
		b.APIVer = ver
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(b *builder) error {
		b.Timeout = &timeout
		return nil
	}
}
