package ollama

import (
	"context"
	"fmt"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ollama/ollama/api"
)

// healthCheck checks the connection to the Ollama server.
// It retries the connection up to MaxRetries times with exponential backoff.
func healthCheck(ctx context.Context, cli *api.Client) error {
	var err error
	for i := 0; i < MaxRetries; i++ {
		if _, err = cli.List(ctx); err == nil {
			return nil
		}
		time.Sleep(min(1<<i*time.Second, MaxRetryWaitingTime))
	}
	return ErrCanNotConnectToServer
}

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
func toOptions(conf any) (map[string]any, error) {
	if conf == nil {
		return nil, nil
	}

	options, ok := conf.(map[string]any)
	if !ok {
		return options, fmt.Errorf("invalid config type: %T, expected %T", conf, *new(map[string]any))
	}
	return options, nil
}
