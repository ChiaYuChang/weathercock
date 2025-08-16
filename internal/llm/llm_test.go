package llm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/llm/gemini"
	"github.com/ChiaYuChang/weathercock/internal/llm/ollama"
	"github.com/ChiaYuChang/weathercock/internal/llm/openai"
	"github.com/stretchr/testify/require"
)

func TestNewPromptTemplate(t *testing.T) {
	template := "Instruct: {{.instruct}}\nQuery: {{.query}}"
	foctory, err := llm.NewPromptTemplateFactory(template)
	require.NoError(t, err)
	require.NotNil(t, foctory)

	query := foctory.NewPromptTemplate(map[string]any{
		"instruct": "Given a web search query, retrieve relevant passages that answer the query",
		"query":    "新北市政府今日宣布，捷運三鶯線的整體工程進度已超過85%，預計最快明年底便可進入通車測試階段。此消息激勵了鶯歌及周邊地區的房市，許多居民期待交通便利性的提升能帶動地方發展，尤其是觀光產業。市長在受訪時表示，這條路線是新北三環六線中的重要一環，市府將全力監督後續工程，確保如期如質完工，為市民帶來更便捷的生活。",
	})
	require.NoError(t, err)
	require.NotNil(t, query)

	qText := query.String()
	require.NotEmpty(t, qText)
	require.Equal(t, template, query.Template())
	require.Contains(t, qText, query.GetVar("instruct"))
	require.Contains(t, qText, query.GetVar("query"))
}

func textGenerateTests(t *testing.T, cli llm.LLM, verbose bool) {
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
			name: "OK",
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
					if verbose {
						t.Log(output)
					}
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

func textEmbedTests(t *testing.T, cli llm.LLM, texts []string, dim int) {
	instruct := "Given a web search query, retrieve relevant passages that answer the query"
	template := "Instruct: {{.instruct}}\nQuery: {{.query}}"
	foctory, err := llm.NewPromptTemplateFactory(template)
	require.NoError(t, err)
	require.NotNil(t, foctory)

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
				n := min(len(texts), 3)
				inputs := make([]llm.EmbedInput, n)
				for i, text := range texts[:n] {
					inputs[i] = foctory.NewPromptTemplate(map[string]any{
						"instruct": instruct,
						"query":    text,
					})
				}

				return &llm.EmbedRequest{Inputs: inputs}
			},
			reqErr: nil,
			testRespFunc: func(t *testing.T, resp *llm.EmbedResponse) {
				for _, embed := range resp.Embeddings {
					require.Equal(t, llm.EmbedStateOk, embed.State)
					require.NotEmpty(t, embed.Values)
					if dim > 0 {
						require.Len(t, embed.Values, dim)
					}
				}

				data, err := json.Marshal(resp)
				require.NoError(t, err)
				require.NotNil(t, data)
			},
		},
		{
			name: "large input",
			genReqFunc: func() *llm.EmbedRequest {
				n := min(len(texts), 100)
				inputs := make([]llm.EmbedInput, n)
				for i, text := range texts[:n] {
					inputs[i] = foctory.NewPromptTemplate(map[string]any{
						"instruct": instruct,
						"query":    text,
					})
				}
				return &llm.EmbedRequest{Inputs: inputs}
			},
			reqErr: nil,
			testRespFunc: func(t *testing.T, resp *llm.EmbedResponse) {
				for _, embed := range resp.Embeddings {
					require.Equal(t, llm.EmbedStateOk, embed.State)
					require.NotEmpty(t, embed.Values)
					if dim > 0 {
						require.Len(t, embed.Values, dim)
					}
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

func TestGeminiGenerate(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not found, skip test")
	}

	var cli llm.LLM
	var err error
	cli, err = gemini.Gemini(
		context.Background(),
		gemini.WithAPIKey(key),
	)

	if err != nil {
		// If we can't connect or the models aren't found, we skip the test.
		t.Skipf("Skipping Gemini tests: could not connect to gemini API or models not found. Error: %v", err)
	}
	require.NotNil(t, cli)
	textGenerateTests(t, cli, true)
}

func TestGeminiEmbed(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not found, skip test")
	}

	var cli llm.LLM
	var err error
	cli, err = gemini.Gemini(
		context.Background(),
		gemini.WithAPIKey(key),
	)

	if err != nil {
		// If we can't connect or the models aren't found, we skip the test.
		t.Skipf("Skipping Gemini tests: could not connect to gemini API or models not found. Error: %v", err)
	}
	require.NotNil(t, cli)

	filename := "./test_embd.txt"
	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.NotNil(t, data)

	texts := strings.Split(string(data), "\n")
	texts = texts[:50] // save tokens
	textEmbedTests(t, cli, texts, 0)
}

func TestOllamaGenerate(t *testing.T) {
	var (
		OllamaHost = "host.docker.internal"
		OllamaPort = 11434
		GenModel   = "gemma3:270m"
		EmbedModel = "bge-large:latest"
	)

	OllamaURL := func() string {
		return fmt.Sprintf("http://%s:%d", OllamaHost, OllamaPort)
	}

	if tmp := os.Getenv("OLLAMA_HOST"); tmp != "" {
		OllamaHost = tmp
	}
	if tmp := os.Getenv("OLLAMA_PORT"); tmp != "" {
		OllamaPort, _ = strconv.Atoi(tmp)
	}

	var cli llm.LLM
	var err error
	cli, err = ollama.Ollama(
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
		t.Skipf("Skipping Ollama tests: could not connect to Ollama server at %s or models not found. Error: %v",
			OllamaURL(), err)
	}
	require.NotNil(t, cli)
	textGenerateTests(t, cli, false)
}

func TestOllamaEmbed(t *testing.T) {
	var (
		OllamaHost = "host.docker.internal"
		OllamaPort = 11434
		GenModel   = "gemma3:270m"
		EmbedModel = "jeffh/intfloat-multilingual-e5-large-instruct:f32"
		EmbedDim   = 1024
	)

	OllamaURL := func() string {
		return fmt.Sprintf("http://%s:%d", OllamaHost, OllamaPort)
	}

	if tmp := os.Getenv("OLLAMA_HOST"); tmp != "" {
		OllamaHost = tmp
	}
	if tmp := os.Getenv("OLLAMA_PORT"); tmp != "" {
		OllamaPort, _ = strconv.Atoi(tmp)
	}

	var cli llm.LLM
	var err error
	cli, err = ollama.Ollama(
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
		t.Skipf("Skipping Ollama tests: could not connect to Ollama server at %s or models not found. Error: %v",
			OllamaURL(), err)
	}
	require.NotNil(t, cli)

	filename := "./test_embd.txt"
	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.NotNil(t, data)

	texts := strings.Split(string(data), "\n")
	textEmbedTests(t, cli, texts, EmbedDim)
}

func TestOpenAIGenerate(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not found, skip test")
	}

	model := "openai/gpt-oss-20b:free"
	baseURL := "https://openrouter.ai/api/v1"

	var cli llm.LLM
	var err error
	cli, err = openai.OpenAI(context.Background(),
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL(baseURL),
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

	if err != nil {
		t.Skipf("Skipping OpenAI tests: could not connect to openai server at %s or models not found. Error: %v",
			baseURL, err)
	}
	require.NotNil(t, cli)
	textGenerateTests(t, cli, false)
}

func TestOpenAIEmbed(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not found, skip test")
	}

	baseURL := "https://openrouter.ai/api/v1"
	dim := 1024

	var cli llm.LLM
	var err error
	cli, err = openai.OpenAI(context.Background(),
		openai.WithAPIKey(key),
		openai.WithMaxRetries(3),
		openai.WithTimeout(30*time.Second),
		openai.WithModel(
			openai.NewOpenAIModel(llm.ModelGenerate, openai.DefaultGenModel),
			openai.NewOpenAIModel(llm.ModelEmbed, openai.DefaultEmbedModel),
		),
		openai.WithDefaultGenerate(openai.DefaultGenModel),
		openai.WithDefaultEmbed(openai.DefaultEmbedModel),
		openai.WithEmbedDim(dim),
	)

	if err != nil {
		t.Skipf("Skipping OpenAI tests: could not connect to openai server at %s or models not found. Error: %v",
			baseURL, err)
	}
	require.NotNil(t, cli)

	filename := "./test_embd.txt"
	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.NotNil(t, data)

	texts := strings.Split(string(data), "\n")
	texts = texts[:50] // save tokens
	textEmbedTests(t, cli, texts, dim)
}
