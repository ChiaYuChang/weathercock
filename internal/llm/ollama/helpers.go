package ollama

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ollama/ollama/api"
)

// healthCheck checks the connection to the Ollama server.
// It retries the connection up to MaxRetries times with exponential backoff.
// Parameters:
//   - ctx: The context for the health check.
//   - cli: The Ollama API client.
//
// Returns:
//   - error: An error if the connection cannot be established after retries.
func healthCheck(ctx context.Context, cli *api.Client) error {
	if cli == nil {
		return ErrOptNilClient
	}

	var err error
	for i := 0; i < MaxRetries; i++ {
		if _, err = cli.List(ctx); err == nil {
			return nil
		}
		time.Sleep(min(1<<i*time.Second, MaxRetryWaitingTime))
	}
	return ErrCanNotConnectToServer
}

// toOllamaMessages converts a slice of llm.Message to a slice of api.Message for Ollama.
// Parameters:
//   - msgs: The slice of llm.Message to convert.
//
// Returns:
//   - []api.Message: The converted slice of Ollama API messages.
func toOllamaMessages(msgs []llm.Message) []api.Message {
	count := 0
	for _, msg := range msgs {
		count += len(msg.Content)
	}

	oMsgs := make([]api.Message, 0, count)
	for _, msg := range msgs {
		role := string(msg.Role)
		for _, content := range msg.Content {
			oMsgs = append(oMsgs, api.Message{
				Role:    role,
				Content: content,
			})
		}
	}
	return oMsgs
}

// toOptions performs a type assertion, returning the result or an error.
// It converts a generic config interface to a map[string]any.
// Parameters:
//   - conf: The configuration to assert.
//
// Returns:
//   - map[string]any: The asserted configuration map.
//   - error: An error if the type assertion fails.
func toOptions(conf any) (map[string]any, error) {
	if conf == nil {
		return nil, nil
	}

	options, ok := conf.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %T, expected %T", ErrInvalidOptionsType, conf, *new(map[string]any))
	}
	return options, nil
}

func extractJSONObject(s string) (string, error) {
	start := strings.Index(s, "{")
	if start == -1 {
		return "", fmt.Errorf("could not find opening brace '{' in the string")
	}

	end := strings.LastIndex(s, "}")
	if end == -1 {
		return "", fmt.Errorf("could not find closing brace '}' in the string")
	}

	if end < start {
		return "", fmt.Errorf("found closing brace '}' before opening brace '{'")
	}

	return s[start : end+1], nil
}
