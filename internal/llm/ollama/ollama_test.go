package ollama_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/llm/ollama"
	"github.com/stretchr/testify/require"
)

// To run these tests, you need a running Ollama instance.
// By default, it connects to http://localhost:11434. You can override this by setting the OLLAMA_HOST environment variable.
// You also need to have the required models pulled:
// `ollama pull llama3`
// `ollama pull nomic-embed-text`

var (
	OllamaHost = "host.docker.internal"
	OllamaPort = 11434
	GenModel   = "gemma3n:e4b"
	// GenModel   = "gpt-oss:20b"
	EmbedModel = "bge-large:latest"
	EmbedDim   = 1024
)

var OllamaURL = fmt.Sprintf("http://%s:%d", OllamaHost, OllamaPort)

func TestOllama(t *testing.T) {
	cli, err := ollama.Ollama(
		context.Background(),
		ollama.WithHost(OllamaURL),
		ollama.WithModel(
			ollama.NewOllamaModel(llm.ModelGenerate, GenModel),
			ollama.NewOllamaModel(llm.ModelEmbed, EmbedModel),
		),
		ollama.WithDefaultGenerate(GenModel),
		ollama.WithDefaultEmbed(EmbedModel),
	)

	if err != nil {
		// If we can't connect or the models aren't found, we skip the test.
		// This is useful for CI environments where Ollama might not be running.
		t.Skipf("Skipping Ollama tests: could not connect to Ollama server at %s or models not found. Error: %v",
			OllamaURL, err)
	}
	require.NotNil(t, cli)

	t.Run("Generate", func(t *testing.T) {
		resp, err := cli.Generate(&llm.GenerateRequest{
			Context: context.Background(),
			Messages: []llm.Message{
				{
					Role: llm.RoleSystem,
					Content: []string{
						"You are a helpful assistant.",
						"You always try to answer users question within 100 words.",
					},
				},
				{
					Role: llm.RoleUser,
					Content: []string{
						"Introduce yourself.",
					},
				},
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Outputs)
		for _, output := range resp.Outputs {
			require.NotEmpty(t, output)
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)
		require.NotNil(t, data)
	})

	t.Run("Embed", func(t *testing.T) {
		resp, err := cli.Embed(&llm.EmbedRequest{
			Ctx: context.Background(),
			Inputs: []llm.EmbedInput{
				llm.NewSimpleText("Hello world"),
				llm.NewSimpleText("Ollama is a great tool for running LLMs locally."),
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Embeddings, 2)
		for _, embed := range resp.Embeddings {
			require.Equal(t, llm.EmbedStateOk, embed.State)
			require.NotEmpty(t, embed.Values)
			require.Len(t, embed.Values, EmbedDim)
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)
		require.NotNil(t, data)
	})
}
