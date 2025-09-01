package gemini_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/llm/gemini"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestGemini(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not found, skip test")
	}

	cli, err := gemini.Gemini(context.Background(),
		gemini.WithAPIKey(key),
		gemini.WithTimeout(30*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, cli)

	t.Run("Generate", func(t *testing.T) {
		resp, err := cli.Generate(
			context.Background(),
			&llm.GenerateRequest{
				Messages: []llm.Message{
					{
						Role:    llm.RoleUser,
						Content: []string{"Please introduce yourself within 50 words."},
					},
				},
			})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Outputs)
		for _, output := range resp.Outputs {
			require.NotEmpty(t, output)
		}
		t.Log(strings.Join(resp.Outputs, "\n"))
	})

	t.Run("Embed", func(t *testing.T) {
		texts := []string{
			"新北市政府今日宣布，捷運三鶯線的整體工程進度已超過85%，預計最快明年底便可進入通車測試階段。此消息激勵了鶯歌及周邊地區的房市，許多居民期待交通便利性的提升能帶動地方發展，尤其是觀光產業。市長在受訪時表示，這條路線是新北三環六線中的重要一環，市府將全力監督後續工程，確保如期如質完工，為市民帶來更便捷的生活。",
			"The Federal Reserve is facing increasing pressure to address inflation, as the latest Consumer Price Index (CPI) report showed a year-over-year increase of 4.5%, exceeding analysts' expectations. While the job market remains strong, rising costs for everyday goods are impacting household budgets across the nation. Economists are now debating whether a more aggressive interest rate hike is necessary, a move that could potentially slow down economic growth but is seen as crucial to curb long-term inflation.",
			"台灣的太空科技新創公司「Galactic Compass」成功發射了其首枚商業觀測衛星「Triton-1」。這枚衛星搭載了高解析度光學儀器，將為農業、環境監測和城市規劃提供精準的數據服務。該公司表示，這次成功的發射證明了台灣在衛星製造及系統整合方面的實力，並已獲得數個來自東南亞國家的合作意向，未來將持續開發更先進的AI影像分析功能。",
		}

		inputs := make([]llm.EmbedInput, len(texts))
		for i, text := range texts {
			inputs[i] = llm.NewSimpleTextInput(text)
		}

		resp, err := cli.Embed(context.Background(),
			&llm.EmbedRequest{
				Inputs: inputs,
			})
		require.NoError(t, err)
		require.Len(t, resp.Embeddings, len(inputs))

		for _, embed := range resp.Embeddings {
			require.Equal(t, llm.EmbedStateOk, embed.State)
			require.NotEmpty(t, embed.Values)
		}
	})
}

func TestGeminiBatchCreate(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not found, skip test")
	}

	model := gemini.DefaultGenModel
	cli, err := gemini.Gemini(context.Background(),
		gemini.WithAPIKey(key),
		gemini.WithTimeout(30*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, cli)

	buf := bytes.NewBuffer(nil)

	resp, err := cli.BatchCreate(context.Background(), &llm.BatchRequest{
		Requests: []llm.Request{
			&llm.GenerateRequest{
				Messages: []llm.Message{
					{
						Role: llm.RoleUser,
						Content: []string{
							"Please introduce yourself within 50 words.",
						},
					},
				},
				ModelName: model,
				Config: &genai.GenerateContentConfig{
					MaxOutputTokens: 60,
				},
			},
			&llm.GenerateRequest{
				Messages: []llm.Message{
					{
						Role: llm.RoleUser,
						Content: []string{
							"Where is the capital of Taiwan?",
						},
					},
				},
				ModelName: model,
				Config: &genai.GenerateContentConfig{
					MaxOutputTokens: 20,
				},
			},
		},
		ReadWriter: buf,
	})

	if err != nil {
		e, ok := err.(genai.APIError)
		require.True(t, ok)
		t.Logf("Code: %d - %s\n", e.Code, e.Status)
		t.Log(e.Message)
		return
	}
	require.NotNil(t, resp)

	data, err := io.ReadAll(buf)
	require.NoError(t, err)
	require.NotNil(t, data)
	if resp.IsDone {
		require.NotEmpty(t, resp.Responses)
		require.Len(t, resp.Responses, 2)
	}
	data, err = json.MarshalIndent(resp, "", "  ")
	require.NoError(t, err)
	t.Log(string(data))

	resp, err = cli.BatchRetrieve(context.Background(), &llm.BatchRetrieveRequest{
		ID: resp.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	data, err = json.MarshalIndent(resp, "", "  ")
	require.NoError(t, err)
	t.Log(string(data))
}

func TestGeminiForamatOutput(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not found, skip test")
	}

	cli, err := gemini.Gemini(context.Background(),
		gemini.WithAPIKey(key),
		gemini.WithTimeout(30*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, cli)

	if err != nil {
		t.Skipf("could not connet to openai, skip test: %v", err)
	}

	require.NotNil(t, cli)

	type Weather struct {
		Index       int    `json:"index"`
		City        string `json:"city"`
		Weather     string `json:"weather"`
		Temperature int    `json:"temperature"`
		Humidity    int    `json:"humidity"`
	}

	type RespFormat struct {
		N       int       `json:"n"`
		Records []Weather `json:"records"`
	}

	data := []Weather{
		{
			City:        "Taipei",
			Weather:     "Sunny",
			Temperature: 25,
			Humidity:    60,
		},
		{
			City:        "London",
			Weather:     "Cloudy",
			Temperature: 15,
			Humidity:    80,
		},
		{
			City:        "New York",
			Weather:     "Rainy",
			Temperature: 10,
			Humidity:    90,
		},
	}
	for i := range data {
		data[i].Index = i + 1
	}

	sb := &strings.Builder{}
	w := csv.NewWriter(sb)
	w.Write([]string{"Index", "City", "Weather", "Temperature", "Humidity"})
	for _, d := range data {
		err := w.Write([]string{
			strconv.Itoa(d.Index),
			d.City,
			d.Weather,
			strconv.Itoa(d.Temperature),
			strconv.Itoa(d.Humidity)})
		require.NoError(t, err)
	}
	w.Flush()

	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := reflector.Reflect(RespFormat{})
	require.NotNil(t, schema)

	prompt := []string{
		"transform the following csv into json format",
		"input: format:",
		"city, weather, temperature, humidity",
		"output format:",
		"{",
		" n: int // number of records",
		" records: [",
		"  {",
		"   index: int,",
		"   city: string,",
		"   weather: string,",
		"   temperature: int,",
		"   humidity: int",
		"  }",
		" ]",
		"}",
	}

	resp, err := cli.Generate(
		context.Background(),
		&llm.GenerateRequest{
			Messages: []llm.Message{
				{
					Role:    llm.RoleSystem,
					Content: prompt,
				},
				{
					Role:    llm.RoleUser,
					Content: []string{sb.String()},
				},
			},
			ModelName: cli.DefaultModels[llm.ModelGenerate],
			Schema: &llm.ResponseSchema{
				Name:   "city_weather",
				Strict: true,
				S:      schema,
			},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Outputs)

	var result RespFormat
	err = json.Unmarshal([]byte(resp.Outputs[0]), &result)
	require.NoError(t, err)
	require.Equal(t, len(data), result.N)
	require.Len(t, result.Records, len(data))

	sort.Slice(result.Records, func(i, j int) bool {
		return result.Records[i].Index < result.Records[j].Index
	})

	for i, r := range result.Records {
		require.Equal(t, data[i], r)
	}
}
