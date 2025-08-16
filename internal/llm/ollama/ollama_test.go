package ollama_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/llm/ollama"
	"github.com/stretchr/testify/require"
)

// To run these tests, you need a running Ollama instance.
// By default, it connects to http://host.docker.internal:11434. You can override this by setting
// the OLLAMA_HOST and OLLAMA_PORT environment variable.
// You also need to have the required models pulled:
//   - for content generatation: `ollama pull gemma3:270m`
//   - for embedding: `ollama pull bge-large:latest`
var (
	OllamaHost = "host.docker.internal"
	OllamaPort = 11434
	GenModel   = "gemma3:270m"
	EmbedModel = "bge-large:latest"
	EmbedDim   = 1024
)

func OllamaURL() string {
	return fmt.Sprintf("http://%s:%d", OllamaHost, OllamaPort)
}

func init() {
	if tmp := os.Getenv("OLLAMA_HOST"); tmp != "" {
		OllamaHost = tmp
	}
	if tmp := os.Getenv("OLLAMA_PORT"); tmp != "" {
		OllamaPort, _ = strconv.Atoi(tmp)
	}
}

func TestOllamaOptions(t *testing.T) {
	tcs := []struct {
		name string
		opts []ollama.Option
		err  error
	}{
		{
			name: "no base url",
			opts: []ollama.Option{
				ollama.WithHTTPClient(http.DefaultClient),
				ollama.WithModel(
					ollama.NewOllamaModel(llm.ModelGenerate, GenModel),
					ollama.NewOllamaModel(llm.ModelEmbed, EmbedModel),
				),
				ollama.WithDefaultGenerate(GenModel),
				ollama.WithDefaultEmbed(EmbedModel),
			},
			err: ollama.ErrNoBaseURL,
		},
		{
			name: "nil http client",
			opts: []ollama.Option{
				ollama.WithHost(OllamaURL()),
				ollama.WithHTTPClient(nil),
				ollama.WithModel(
					ollama.NewOllamaModel(llm.ModelGenerate, GenModel),
					ollama.NewOllamaModel(llm.ModelEmbed, EmbedModel),
				),
				ollama.WithDefaultGenerate(GenModel),
				ollama.WithDefaultEmbed(EmbedModel),
			},
			err: ollama.ErrOptNilClient,
		},
		{
			name: "with http client",
			opts: []ollama.Option{
				ollama.WithHost(OllamaURL()),
				ollama.WithHTTPClient(http.DefaultClient),
				ollama.WithModel(
					ollama.NewOllamaModel(llm.ModelGenerate, GenModel),
					ollama.NewOllamaModel(llm.ModelEmbed, EmbedModel),
				),
				ollama.WithDefaultGenerate(GenModel),
				ollama.WithDefaultEmbed(EmbedModel),
			},
			err: nil,
		},
		{
			name: "add the same model twice",
			opts: []ollama.Option{
				ollama.WithHost(OllamaURL()),
				ollama.WithHTTPClient(http.DefaultClient),
				ollama.WithModel(
					ollama.NewOllamaModel(llm.ModelGenerate, GenModel),
					ollama.NewOllamaModel(llm.ModelGenerate, GenModel),
				),
				ollama.WithDefaultGenerate(GenModel),
				ollama.WithDefaultEmbed(EmbedModel),
			},
			err: ollama.ErrOptModelHasAreadyAdded,
		},
		{
			name: "no default model",
			opts: []ollama.Option{
				ollama.WithHost(OllamaURL()),
				ollama.WithHTTPClient(http.DefaultClient),
				ollama.WithModel(
					ollama.NewOllamaModel(llm.ModelGenerate, GenModel),
					ollama.NewOllamaModel(llm.ModelEmbed, EmbedModel),
				),
			},
			err: ollama.ErrNoDefaultModel,
		},
		{
			name: "no default model",
			opts: []ollama.Option{
				ollama.WithHost(OllamaURL()),
				ollama.WithHTTPClient(http.DefaultClient),
				ollama.WithModel(
					ollama.NewOllamaModel(llm.ModelGenerate, EmbedModel),
					ollama.NewOllamaModel(llm.ModelEmbed, GenModel),
				),
				ollama.WithDefaultGenerate(EmbedModel),
				ollama.WithDefaultEmbed(GenModel),
			},
			err: ollama.ErrModelNotSupport,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ollama.Ollama(context.Background(), tc.opts...)
			if errors.Is(err, ollama.ErrCanNotConnectToServer) {
				t.Skip("can not connect to server, test skipped")
			}

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOllamaGenerate(t *testing.T) {
	cli, err := ollama.Ollama(
		context.Background(),
		ollama.WithHost(OllamaURL()),
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
			OllamaURL(), err)
	}
	require.NotNil(t, cli)

	tcs := []struct {
		name         string
		genReqFunc   func() *llm.GenerateRequest
		reqErr       error
		testRespFunc func(t *testing.T, resp *llm.GenerateResponse)
	}{
		{
			name: "nil request",
			genReqFunc: func() *llm.GenerateRequest {
				return nil
			},
			reqErr: llm.ErrRequestShouldNotBeNull,
			testRespFunc: func(t *testing.T, resp *llm.GenerateResponse) {
				require.Nil(t, resp)
			},
		},
		{
			name: "no input",
			genReqFunc: func() *llm.GenerateRequest {
				return &llm.GenerateRequest{}
			},
			reqErr: llm.ErrNoInput,
			testRespFunc: func(t *testing.T, resp *llm.GenerateResponse) {
				require.Nil(t, resp)
			},
		},
		{
			name: "OK w/o options",
			genReqFunc: func() *llm.GenerateRequest {
				return &llm.GenerateRequest{
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
				}
			},
			reqErr: nil,
			testRespFunc: func(t *testing.T, resp *llm.GenerateResponse) {
				require.NotEmpty(t, resp.Outputs)
				for _, output := range resp.Outputs {
					require.NotEmpty(t, output)
				}

				data, err := json.Marshal(resp)
				require.NoError(t, err)
				require.NotNil(t, data)
			},
		},
		{
			name: "OK with options",
			genReqFunc: func() *llm.GenerateRequest {
				return &llm.GenerateRequest{
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
								"What is the capital of France?",
							},
						},
					},
					Config: map[string]any{
						"temperature": 0.7,
					},
				}
			},
			reqErr: nil,
			testRespFunc: func(t *testing.T, resp *llm.GenerateResponse) {
				require.NotEmpty(t, resp.Outputs)
				for _, output := range resp.Outputs {
					require.NotEmpty(t, output)
				}

				data, err := json.Marshal(resp)
				require.NoError(t, err)
				require.NotNil(t, data)
			},
		},
		{
			name: "with wrong option type",
			genReqFunc: func() *llm.GenerateRequest {
				return &llm.GenerateRequest{
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
								"What is the capital of France?",
							},
						},
					},
					Config: map[string]float32{
						"temperature": 0.7,
					},
				}
			},
			reqErr: ollama.ErrInvalidOptionsType,
			testRespFunc: func(t *testing.T, resp *llm.GenerateResponse) {
				require.Nil(t, resp)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.genReqFunc()
			resp, err := cli.Generate(context.Background(), req)
			if tc.reqErr != nil {
				require.ErrorIs(t, err, tc.reqErr)
				return
			}
			require.NotNil(t, resp)
			tc.testRespFunc(t, resp)
		})
	}
}

func TestOllamaEmbed(t *testing.T) {
	cli, err := ollama.Ollama(
		context.Background(),
		ollama.WithHost(OllamaURL()),
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
			OllamaURL(), err)
	}
	require.NotNil(t, cli)

	tcs := []struct {
		name         string
		genReqFunc   func() *llm.EmbedRequest
		reqErr       error
		testRespFunc func(t *testing.T, resp *llm.EmbedResponse)
	}{
		{
			name: "nil request",
			genReqFunc: func() *llm.EmbedRequest {
				return nil
			},
			reqErr: llm.ErrRequestShouldNotBeNull,
			testRespFunc: func(t *testing.T, resp *llm.EmbedResponse) {
				require.Nil(t, resp)
			},
		},
		{
			name: "no input",
			genReqFunc: func() *llm.EmbedRequest {
				return &llm.EmbedRequest{}
			},
			reqErr: llm.ErrNoInput,
			testRespFunc: func(t *testing.T, resp *llm.EmbedResponse) {
				require.Nil(t, resp)
			},
		},
		{
			name: "small input",
			genReqFunc: func() *llm.EmbedRequest {
				texts := []string{
					"新北市政府今日宣布，捷運三鶯線的整體工程進度已超過85%，預計最快明年底便可進入通車測試階段。此消息激勵了鶯歌及周邊地區的房市，許多居民期待交通便利性的提升能帶動地方發展，尤其是觀光產業。市長在受訪時表示，這條路線是新北三環六線中的重要一環，市府將全力監督後續工程，確保如期如質完工，為市民帶來更便捷的生活。",
					"The Federal Reserve is facing increasing pressure to address inflation, as the latest Consumer Price Index (CPI) report showed a year-over-year increase of 4.5%, exceeding analysts' expectations. While the job market remains strong, rising costs for everyday goods are impacting household budgets across the nation. Economists are now debating whether a more aggressive interest rate hike is necessary, a move that could potentially slow down economic growth but is seen as crucial to curb long-term inflation.",
					"台灣的太空科技新創公司「Galactic Compass」成功發射了其首枚商業觀測衛星「Triton-1」。這枚衛星搭載了高解析度光學儀器，將為農業、環境監測和城市規劃提供精準的數據服務。該公司表示，這次成功的發射證明了台灣在衛星製造及系統整合方面的實力，並已獲得數個來自東南亞國家的合作意向，未來將持續開發更先進的AI影像分析功能。",
				}

				inputs := make([]llm.EmbedInput, len(texts))
				for i, text := range texts {
					inputs[i] = llm.NewSimpleTextInput(text)
				}

				return &llm.EmbedRequest{Inputs: inputs}
			},
			reqErr: nil,
			testRespFunc: func(t *testing.T, resp *llm.EmbedResponse) {
				for _, embed := range resp.Embeddings {
					require.Equal(t, llm.EmbedStateOk, embed.State)
					require.NotEmpty(t, embed.Values)
					require.Len(t, embed.Values, EmbedDim)
				}

				data, err := json.Marshal(resp)
				require.NoError(t, err)
				require.NotNil(t, data)
			},
		},
		{
			name: "large input",
			genReqFunc: func() *llm.EmbedRequest {
				filename := "../test_embd.txt"
				data, err := os.ReadFile(filename)
				require.NoError(t, err)
				require.NotNil(t, data)

				texts := strings.Split(string(data), "\n")
				texts = texts[:10]
				inputs := make([]llm.EmbedInput, len(texts))
				for i, text := range texts {
					inputs[i] = llm.NewSimpleTextInput(text)
				}
				return &llm.EmbedRequest{Inputs: inputs}
			},
			reqErr: nil,
			testRespFunc: func(t *testing.T, resp *llm.EmbedResponse) {
				for _, embed := range resp.Embeddings {
					require.Equal(t, llm.EmbedStateOk, embed.State)
					require.NotEmpty(t, embed.Values)
					require.Len(t, embed.Values, EmbedDim)
				}

				data, err := json.Marshal(resp)
				require.NoError(t, err)
				require.NotNil(t, data)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.genReqFunc()
			resp, err := cli.Embed(context.Background(), req)
			if tc.reqErr != nil {
				require.ErrorIs(t, err, tc.reqErr)
				return
			}
			require.Len(t, resp.Embeddings, len(req.Inputs))
			require.NotNil(t, resp)
			tc.testRespFunc(t, resp)
		})
	}
}
