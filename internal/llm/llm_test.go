package llm_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/llm/gemini"
	"github.com/ChiaYuChang/weathercock/internal/llm/ollama"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestGemini(t *testing.T) {
	var cli llm.LLM
	var err error
	cli, err = gemini.Gemini(
		context.Background(),
		gemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
	)

	if err != nil {
		// If we can't connect or the models aren't found, we skip the test.
		t.Skipf("Skipping Gemini tests: could not connect to gemini API or models not found. Error: %v", err)
	}
	require.NotNil(t, cli)

	t.Run("Generate", func(t *testing.T) {
		resp, err := cli.Generate(&llm.GenerateRequest{
			Context: context.Background(),
			Messages: []llm.Message{
				{
					Role: llm.RoleSystem,
					Content: []string{
						"你是一個笑話大師，擅長依據觀眾需求回應有趣的笑話",
					},
				},
				{
					Role: llm.RoleUser,
					Content: []string{
						"請說一個生物學相關的笑話",
						"請用繁體中文回答",
					},
				},
			},
		})
		require.NoError(t, err)
		t.Log(resp.Outputs)
	})

	t.Run("Embed", func(t *testing.T) {
		var dim int32 = 1024
		inputs := []llm.EmbedInput{
			llm.NewSimpleText("Gemini是由Google開發的生成式人工智慧聊天機器人。它基於同名的Gemini系列大型語言模型。是應對OpenAI公司開發的ChatGPT聊天機器人的崛起而開發的。其在2023年3月以有限的規模推出，2023年5月擴展到更多個國家。2024年2月8日從Bard更名為Gemini。"),
			llm.NewSimpleText("Gemini CLI 在程式編寫方面表現出色，但它的功能遠不止於此。它是一款多功能、本機運作的工具，協助開發者處理從內容生成、問題解決，到深入研究和任務管理等各種任務。"),
			llm.NewSimpleText("The Google Gen AI SDK provides a unified interface to Gemini 2.5 Pro and Gemini 2.0 models through both the Gemini Developer API and the Gemini API on Vertex AI. With a few exceptions, code that runs on one platform will run on both. This means that you can prototype an application using the Gemini Developer API and then migrate the application to Vertex AI without rewriting your code."),
		}

		resp, err := cli.Embed(&llm.EmbedRequest{
			Ctx:    context.Background(),
			Inputs: inputs,
			Config: &genai.EmbedContentConfig{
				TaskType:             gemini.EmbedTaskRetrivalDocument,
				OutputDimensionality: &dim,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Embeddings, len(inputs))
		for _, embed := range resp.Embeddings {
			require.Equal(t, llm.EmbedStateOk, embed.State)
			require.Equal(t, int(dim), embed.Dim())
			t.Log(embed)
		}
	})
}

func TestOllama(t *testing.T) {
	var (
		OllamaHost = "host.docker.internal"
		OllamaPort = 11434
		GenModel   = "gemma3n:e4b"
		EmbedModel = "bge-large:latest"
		EmbedDim   = 1024
	)
	var OllamaURL = fmt.Sprintf("http://%s:%d", OllamaHost, OllamaPort)

	var cli llm.LLM
	var err error
	cli, err = ollama.Ollama(
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
	})

	t.Run("Embed", func(t *testing.T) {
		data, err := os.ReadFile("./test_embd.txt")
		require.NoError(t, err)
		require.NotNil(t, data)

		lines := strings.Split(string(data), "\n")
		inputs := make([]llm.EmbedInput, len(lines))
		for i, line := range lines {
			inputs[i] = llm.NewSimpleText(line)
		}

		resp, err := cli.Embed(&llm.EmbedRequest{
			Ctx:       context.Background(),
			Inputs:    inputs,
			ModelName: EmbedModel,
		})
		require.NoError(t, err)
		require.Len(t, resp.Embeddings, len(inputs))
		for _, embed := range resp.Embeddings {
			require.Equal(t, llm.EmbedStateOk, embed.State)
			require.NotEmpty(t, embed.Values)
			require.Equal(t, embed.State, llm.EmbedStateOk)
			require.Equal(t, EmbedDim, embed.Dim())
		}
	})
}
