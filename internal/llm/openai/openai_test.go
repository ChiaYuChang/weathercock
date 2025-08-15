package openai_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/llm/openai"
	"github.com/stretchr/testify/require"
)

func TestOpenRouter(t *testing.T) {
	model := "openai/gpt-oss-20b:free"
	cli, err := openai.OpenAI(context.Background(),
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel(
			openai.NewOpenAIModel(llm.ModelGenerate, model),
			openai.NewOpenAIModel(llm.ModelEmbed, openai.DefaultEmbedModel),
		),
		openai.WithDefaultGenerate(model),
		openai.WithDefaultEmbed(openai.DefaultEmbedModel),
		openai.WithMaxRetries(3),
		openai.WithTimeout(30*time.Second),
		openai.UseChatChatCompletions(),
	)
	require.NoError(t, err)
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
						"Please introduce yourself within 100 words.",
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

		t.Log(strings.Join(resp.Outputs, "\n"))
	})
}

func TestOpenAI(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not found, skip test")
	}

	embedDim := 1024
	cli, err := openai.OpenAI(context.Background(),
		openai.WithAPIKey(key),
		openai.WithMaxRetries(3),
		openai.WithTimeout(30*time.Second),
		openai.WithModel(
			openai.NewOpenAIModel(llm.ModelGenerate, openai.DefaultGenModel),
			openai.NewOpenAIModel(llm.ModelEmbed, openai.DefaultEmbedModel),
		),
		openai.WithDefaultGenerate(openai.DefaultGenModel),
		openai.WithDefaultEmbed(openai.DefaultEmbedModel),
		openai.WithEmbedDim(embedDim),
	)

	if err != nil {
		t.Skipf("could not connet to openai, skip test: %v", err)
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
						"Please introduce yourself within 100 words.",
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
		texts := []string{
			"新北市政府今日宣布，捷運三鶯線的整體工程進度已超過85%，預計最快明年底便可進入通車測試階段。此消息激勵了鶯歌及周邊地區的房市，許多居民期待交通便利性的提升能帶動地方發展，尤其是觀光產業。市長在受訪時表示，這條路線是新北三環六線中的重要一環，市府將全力監督後續工程，確保如期如質完工，為市民帶來更便捷的生活。",
			"The Federal Reserve is facing increasing pressure to address inflation, as the latest Consumer Price Index (CPI) report showed a year-over-year increase of 4.5%, exceeding analysts' expectations. While the job market remains strong, rising costs for everyday goods are impacting household budgets across the nation. Economists are now debating whether a more aggressive interest rate hike is necessary, a move that could potentially slow down economic growth but is seen as crucial to curb long-term inflation.",
			"台灣的太空科技新創公司「Galactic Compass」成功發射了其首枚商業觀測衛星「Triton-1」。這枚衛星搭載了高解析度光學儀器，將為農業、環境監測和城市規劃提供精準的數據服務。該公司表示，這次成功的發射證明了台灣在衛星製造及系統整合方面的實力，並已獲得數個來自東南亞國家的合作意向，未來將持續開發更先進的AI影像分析功能。",
		}

		inputs := make([]llm.EmbedInput, len(texts))
		for i, text := range texts {
			inputs[i] = llm.NewSimpleText(text)
		}

		resp, err := cli.Embed(&llm.EmbedRequest{
			Ctx:    context.Background(),
			Inputs: inputs,
		})
		require.NoError(t, err)
		require.Len(t, resp.Embeddings, len(inputs))

		data, err := json.Marshal(resp)
		require.NoError(t, err)
		require.NotNil(t, data)

		for _, embed := range resp.Embeddings {
			require.Equal(t, llm.EmbedStateOk, embed.State)
			require.NotEmpty(t, embed.Values)
			require.Len(t, embed.Values, embedDim)
		}
	})
}
