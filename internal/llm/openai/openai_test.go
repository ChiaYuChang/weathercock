package openai_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	openaiplug "github.com/ChiaYuChang/weathercock/internal/llm/openai"
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v2"
	"github.com/stretchr/testify/require"
)

func TestOpanAIUseChatCompletions(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not found, skip test")
	}

	model := "gpt-5-nano"
	cli, err := openaiplug.OpenAI(context.Background(),
		openaiplug.WithAPIKey(key),
		openaiplug.WithModel(
			openaiplug.NewOpenAIModel(llm.ModelGenerate, model),
			openaiplug.NewOpenAIModel(llm.ModelEmbed, openaiplug.DefaultEmbedModel),
		),
		openaiplug.WithDefaultGenerate(model),
		openaiplug.WithDefaultEmbed(openaiplug.DefaultEmbedModel),
		openaiplug.WithMaxRetries(3),
		openaiplug.WithTimeout(30*time.Second),
		openaiplug.UseChatChatCompletions(),
	)
	require.NoError(t, err)
	require.NotNil(t, cli)

	t.Run("Generate", func(t *testing.T) {
		resp, err := cli.Generate(
			context.Background(),
			&llm.GenerateRequest{
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

func TestOpenAIUseRequest(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not found, skip test")
	}

	embedDim := 1024
	cli, err := openaiplug.OpenAI(context.Background(),
		openaiplug.WithAPIKey(key),
		openaiplug.WithMaxRetries(3),
		openaiplug.WithTimeout(30*time.Second),
		openaiplug.WithModel(
			openaiplug.NewOpenAIModel(llm.ModelGenerate, openaiplug.DefaultGenModel),
			openaiplug.NewOpenAIModel(llm.ModelEmbed, openaiplug.DefaultEmbedModel),
		),
		openaiplug.WithDefaultGenerate(openaiplug.DefaultGenModel),
		openaiplug.WithDefaultEmbed(openaiplug.DefaultEmbedModel),
		openaiplug.WithEmbedDim(embedDim),
	)

	if err != nil {
		t.Skipf("could not connet to openai, skip test: %v", err)
	}

	require.NotNil(t, cli)

	t.Run("Generate", func(t *testing.T) {
		resp, err := cli.Generate(
			context.Background(),
			&llm.GenerateRequest{
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
			inputs[i] = llm.NewSimpleTextInput(text)
		}

		resp, err := cli.Embed(context.Background(),
			&llm.EmbedRequest{
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

func TestOpenAIBatchCreate(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not found, skip test")
	}

	embedDim := 16
	cli, err := openaiplug.OpenAI(context.Background(),
		openaiplug.WithAPIKey(key),
		openaiplug.WithMaxRetries(3),
		openaiplug.WithTimeout(30*time.Second),
		openaiplug.WithModel(
			openaiplug.NewOpenAIModel(llm.ModelGenerate, openaiplug.DefaultGenModel),
			openaiplug.NewOpenAIModel(llm.ModelEmbed, openaiplug.DefaultEmbedModel),
		),
		openaiplug.WithDefaultGenerate(openaiplug.DefaultGenModel),
		openaiplug.WithDefaultEmbed(openaiplug.DefaultEmbedModel),
		openaiplug.WithEmbedDim(embedDim),
	)

	if err != nil {
		t.Skipf("could not connet to openai, skip test: %v", err)
	}
	require.NotNil(t, cli)

	wd, err := os.Getwd()
	require.NoError(t, err)
	filename := filepath.Join(wd, "internal", "llm", "test_embd.txt")
	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.NotNil(t, data)

	texts := strings.Split(string(data), "\n")
	require.NotEmpty(t, texts)
	reqs := make([]llm.Request, 0, len(texts)/10+1)
	for i, j := 0, 0; i < len(texts); i++ {
		inputs := []llm.EmbedInput{}
		for j = i; j < i+10 && j < len(texts); j++ {
			inputs = append(inputs, llm.NewSimpleTextInput(texts[j]))
		}
		i = j - 1
		reqs = append(reqs, &llm.EmbedRequest{
			Inputs:    inputs,
			ModelName: cli.DefaultModels[llm.ModelEmbed],
		})
	}

	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "test_*.txt")
	require.NoError(t, err)
	require.NotNil(t, f)
	defer f.Close()

	resp, err := cli.BatchCreate(context.Background(),
		&llm.BatchRequest{
			ModelName:    cli.DefaultModels[llm.ModelEmbed],
			Endpoint:     string(openai.BatchNewParamsEndpointV1Embeddings),
			BatchJobName: "openai-batch-test-file",
			Requests:     reqs,
			ReadWriter:   f,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, resp)

	raw, ok := resp.Raw.(map[string]any)
	require.True(t, ok)
	require.NotNil(t, raw)

	f.Seek(0, io.SeekStart)
	jsonl, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NotNil(t, jsonl)
	jsonl = bytes.TrimSpace(jsonl)

	lines := bytes.Split(jsonl, []byte("\n"))
	require.NotEmpty(t, lines)
	require.Len(t, lines, len(reqs))

	file, ok := raw["file"].(*openai.FileObject)
	require.True(t, ok)
	require.NotNil(t, file)

	require.Contains(t, file.Filename, "openai-batch-test")
	require.Equal(t, file.Purpose, openai.FileObjectPurposeBatch)

	batch, ok := raw["batch"].(*openai.Batch)
	require.True(t, ok)
	require.NotNil(t, batch)

	require.Contains(t, file.Filename, "openai-batch-test")
	require.Equal(t, file.Purpose, openai.FileObjectPurposeBatch)
}

func TestOpenAIBatchRetrive(t *testing.T) {
	tcs := []struct {
		Name            string
		HTTPStatus      int
		RetrieveErrMsg  string
		BatchID         string
		BatchStatusFile [][2]string
		BatchResultFile string
	}{
		{
			Name:           "Batch_Not_Found",
			HTTPStatus:     http.StatusNotFound,
			RetrieveErrMsg: "missing required batch_id parameter",
		},
		{
			Name:       "Batch_Cacelled",
			HTTPStatus: http.StatusOK,
			BatchID:    "batch_68a49aeda4f881908e42162e5d37453e",
			BatchStatusFile: [][2]string{
				{
					string(openai.BatchStatusCancelled),
					"./68a49_batch_cancelled_status.json",
				},
			},
			BatchResultFile: "./68a49_batch_cancelled_results.jsonl",
		},
		{
			Name:       "Batch_Completed",
			HTTPStatus: http.StatusOK,
			BatchID:    "batch_68a5f3a332788190a219f8b62fe1ed33",
			BatchStatusFile: [][2]string{
				{
					string(openai.BatchStatusInProgress),
					"./68a5f_batch_inprogress_status.json",
				},
				{
					string(openai.BatchStatusCompleted),
					"./68a5f_batch_completed_status.json",
				},
			},
			BatchResultFile: "./68a5f_batch_completed_results.jsonl",
		},
	}

	sCode := map[string]int{}
	for _, tc := range tcs {
		sCode[tc.BatchID] = tc.HTTPStatus
	}

	status := map[string][][]byte{}
	result := map[string][]byte{}
	re := map[string]*regexp.Regexp{
		string(openai.BatchStatusCompleted): regexp.MustCompile("\"output_file_id\": \"(file-\\w+)\""),
		string(openai.BatchStatusCancelled): regexp.MustCompile("\"error_file_id\": \"(file-\\w+)\""),
	}

	for _, tc := range tcs {
		if len(tc.BatchStatusFile) <= 0 {
			continue
		}

		wd, err := os.Getwd()
		require.NoError(t, err)
		for _, bsf := range tc.BatchStatusFile {
			filePath := filepath.Join(wd, "internal", "llm", "openai", bsf[1])
			s, err := os.ReadFile(filePath)
			require.NoError(t, err)
			require.NotNil(t, s)
			status[tc.BatchID] = append(status[tc.BatchID], s)
			if openaiplug.IsTerminalJobState(openai.BatchStatus(bsf[0])) {
				re := re[bsf[0]]
				subm := re.FindSubmatch(s)
				require.Len(t, subm, 2)

				fID := string(subm[1])
				resultFilePath := filepath.Join(wd, "internal", "llm", "openai", tc.BatchResultFile)
				r, err := os.ReadFile(resultFilePath)
				require.NoError(t, err)
				result[fID] = r
			}
		}
	}

	key := "my-openai-key"
	ithQuery := map[string]int{}

	mux := http.NewServeMux()
	mux.HandleFunc("/batches/{batch_id}", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bID := r.PathValue("batch_id")
		i := ithQuery[bID]
		ithQuery[bID] = min(i+1, len(status[bID]))
		data, ok := status[bID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			m := map[string]any{
				"message": fmt.Sprintf("No batch found with id '%s'.", bID),
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    nil,
			}
			data, _ := json.Marshal(m)
			w.Write(data)
			return
		}
		require.NotNil(t, data)
		w.Header().Set("Content-Type", "application/json")
		c, ok := sCode[bID]
		require.True(t, ok)

		w.WriteHeader(c)
		w.Write(data[i])
	})

	mux.HandleFunc("/files/{file_id}/content", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		fID := r.PathValue("file_id")
		data, ok := result[fID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(map[string]any{
			"object": "list",
			"data": []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				OwnedBy string `json:"owned_by"`
				Created int64  `json:"created"`
			}{
				{
					ID:      "gpt-4-0613",
					Object:  "model",
					OwnedBy: "openai",
					Created: 1686588896,
				},
				{
					ID:      "gpt-4",
					Object:  "model",
					OwnedBy: "openai",
					Created: 1687882411,
				},
				{
					ID:      "gpt-5-nano",
					Object:  "model",
					OwnedBy: "system",
					Created: 1754426384,
				},
				{
					ID:      "text-embedding-3-small",
					Object:  "model",
					OwnedBy: "system",
					Created: 1705948997,
				},
			},
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dim := 1024
	cli, err := openaiplug.OpenAI(context.Background(),
		openaiplug.WithAPIKey(key),
		openaiplug.WithMaxRetries(3),
		openaiplug.WithTimeout(30*time.Second),
		openaiplug.WithBaseURL(server.URL),
		openaiplug.WithModel(
			openaiplug.NewOpenAIModel(llm.ModelGenerate, openaiplug.DefaultGenModel),
			openaiplug.NewOpenAIModel(llm.ModelEmbed, openaiplug.DefaultEmbedModel),
		),
		openaiplug.WithDefaultGenerate(openaiplug.DefaultGenModel),
		openaiplug.WithDefaultEmbed(openaiplug.DefaultEmbedModel),
		openaiplug.WithEmbedDim(dim),
	)
	require.NoError(t, err)
	require.NotNil(t, cli)
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			var resp = &llm.BatchResponse{}
			var err error

			retry := 0
			for !openaiplug.IsTerminalJobState(openai.BatchStatus(resp.Status)) {
				resp, err = cli.BatchRetrieve(context.Background(), &llm.BatchRetrieveRequest{
					ID: tc.BatchID,
				})
				if tc.RetrieveErrMsg != "" {
					require.ErrorContains(t, err, tc.RetrieveErrMsg)
					return
				}

				require.NoError(t, err)
				require.NotNil(t, resp)

				i := max(0, ithQuery[tc.BatchID]-1)
				require.Equal(t, string(tc.BatchStatusFile[i][0]), resp.Status)
				if resp.IsDone {
					break
				}
				time.Sleep(time.Duration(min(1<<retry, 10)) * time.Second)
				retry++
			}
			require.Equal(t, retry, len(tc.BatchStatusFile)-1)
		})
	}
}

func TestOpenAIForamatOutput(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not found, skip test")
	}

	embedDim := 1024
	cli, err := openaiplug.OpenAI(context.Background(),
		openaiplug.WithAPIKey(key),
		openaiplug.WithMaxRetries(3),
		openaiplug.WithTimeout(30*time.Second),
		openaiplug.WithModel(
			openaiplug.NewOpenAIModel(llm.ModelGenerate, openaiplug.DefaultGenModel),
			openaiplug.NewOpenAIModel(llm.ModelEmbed, openaiplug.DefaultEmbedModel),
		),
		openaiplug.WithDefaultGenerate(openaiplug.DefaultGenModel),
		openaiplug.WithDefaultEmbed(openaiplug.DefaultEmbedModel),
		openaiplug.WithEmbedDim(embedDim),
	)

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

func LogJson(t *testing.T, v any) {
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NotNil(t, data)
	t.Log(string(data))
}
