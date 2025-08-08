package llm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestChunckOffsets(t *testing.T) {
	raw, err := os.ReadFile("./test_text001.txt")
	require.NoError(t, err, "Failed to read test text file")
	text := string(raw)
	require.NotEmpty(t, text, "Test text should not be empty")

	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}
	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				offsets, err := llm.ChunckOffsets(text, tc.ChunkSize, tc.Overlap)
				require.NoError(t, err, "ChunckOffsets should not return an error")
				builder := strings.Builder{}
				for _, off := range offsets {
					_, _, unique, _ := llm.ExtractChunk(text, off)
					builder.WriteString(unique)
				}
				require.Equal(t, text, builder.String(), "Rebuilt text should match original text")
			},
		)
	}
}

func TestChunckParagraphsOffsets(t *testing.T) {
	raw, err := os.ReadFile("./test_text002.txt")
	require.NoError(t, err, "Failed to read test text file")

	paragraphs := strings.Split(string(raw), "\n\n")
	require.NotEmpty(t, paragraphs, "Test paragraphs should not be empty")
	for i, p := range paragraphs {
		paragraphs[i] = strings.TrimSpace(p)
	}

	article := strings.Join(paragraphs, "")
	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}
	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				offsets, err := llm.ChunckParagraphsOffsets(paragraphs, tc.ChunkSize, tc.Overlap)
				require.NoError(t, err, "ChunckParagraphsOffsets should not return an error")
				builder := strings.Builder{}
				for _, off := range offsets {
					_, _, unique, _ := llm.ExtractChunk(article, off)
					builder.WriteString(unique)
				}
				require.Equal(t, article, builder.String(), "Rebuilt text should match original paragraphs")
			},
		)
	}
}

func TestChuncking(t *testing.T) {
	raw, err := os.ReadFile("./test_text001.txt")
	require.NoError(t, err, "Failed to read test text file")
	text := string(raw)
	require.NotEmpty(t, text, "Test text should not be empty")

	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 4096, Overlap: 512},
		{ChunkSize: 1024, Overlap: 128},
		{ChunkSize: 512, Overlap: 64},
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}

	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				chunks, err := llm.Chunck(text, tc.ChunkSize, tc.Overlap)
				require.NoError(t, err, "Chuncking should not return an error")
				rebuild := strings.Builder{}
				for _, chunk := range chunks {
					rebuild.WriteString(chunk[1])
				}
				require.Equal(t, text, rebuild.String(), "Rebuilt text should match original text")
			},
		)
	}

	// Test with invalid parameters
	_, err = llm.Chunck("", 0, 0)
	require.Error(t, err, "Chuncking with invalid parameters should return an error")

	_, err = llm.Chunck(text, 32, 32)
	require.Error(t, err, "Chuncking with invalid overlap should return an error")
}

func TestChunckParagraphs(t *testing.T) {
	raw, err := os.ReadFile("./test_text002.txt")
	require.NoError(t, err, "Failed to read test text file")

	paragraphs := strings.Split(string(raw), "\n\n")
	require.NotEmpty(t, paragraphs, "Test paragraphs should not be empty")
	for i, p := range paragraphs {
		paragraphs[i] = strings.TrimSpace(p)
	}

	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}
	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				chunks, err := llm.ChunckParagraphs(paragraphs, 64, 16)
				require.NoError(t, err, "Chuncking paragraphs should not return an error")
				builder := strings.Builder{}
				for _, chunk := range chunks {
					builder.WriteString(chunk[1])
				}
				require.Equal(t, strings.Join(paragraphs, ""), builder.String(), "Rebuilt text should match original paragraphs")
			},
		)
	}
}

func TestGemini(t *testing.T) {
	apikey, err := os.ReadFile("/home/cychang/Private/Gemini")
	require.NoError(t, err, "Failed to read Gemini API key file")
	apikey = bytes.TrimSpace(apikey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cli, err := llm.NewGemini(ctx, &genai.ClientConfig{
		APIKey:  string(apikey),
		Backend: genai.BackendGeminiAPI,
	})
	require.NoError(t, err, "Failed to create Gemini client")
	require.NotNil(t, cli, "Gemini client should not be nil")

	chat := cli.ChatCompletion()

	user := "user-123"
	model := "gemini-2.5-flash-lite-preview-06-17"
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"answer": map[string]interface{}{
				"type":        "string",
				"description": "The answer to the user's question",
			},
		},
		"required":             []string{"answer"},
		"additionalProperties": false,
	}

	resp, err := chat.New(ctx, model, &llm.ChatCompletionRequest{
		Model: model,
		Messages: []llm.ChatCompletionMessage{
			llm.NewSystemMessage(
				"",
				"you are a teacher. You answer questions in a concise and clear manner.",
				"input format: {\"question\": \"your question\"}",
				"output format: {\"answer\": \"your answer\"}",
			),
			llm.NewUserMessage(
				user,
				"{\"question\": \"What is the capital of Taiwan?\"}",
			),
		},
		ModelOptions: llm.ModelOptions{
			Temperature:     1.0,
			MaxOutputTokens: 1024,
		},
		Schema: schema,
	})
	require.NoError(t, err, "Chat completion should not return an error")
	require.NotNil(t, resp, "Chat completion response should not be nil")
	require.NotEmpty(t, resp.ID, "Chat completion response ID should not be empty")

	msg := resp.Messages[0]
	require.NotEmpty(t, msg.Content(), "Chat completion message content should not be empty")
	t.Logf("Role=%s, Content=%s", msg.Role(), msg.Content())

	ans := map[string]string{}
	err = json.NewDecoder(strings.NewReader(msg.Content()[0])).Decode(&ans)
	require.NoError(t, err, "Failed to decode chat completion message content")
	require.NotEmpty(t, ans["answer"], "Chat completion answer should not be empty")
	t.Logf("Answer: %s", ans["answer"])
}

// func TestOllamaChat(t *testing.T) {
// 	prompt := `You are a keyword extraction expert specializing in Traditional Chinese news articles. Your task is to extract meaningful keywords that best represent the article’s main themes, significant entities, and key actions or concepts.
// Guidelines:
// 1. Context and Relevance:
//     - Focus on cultural and linguistic nuances specific to Traditional Chinese, especially as used in Taiwan.
//     - Prioritize region-specific terms, idiomatic expressions, and phrases relevant to Taiwanese perspectives, policies, and issues.
//     - Prioritize keywords relevant to Taiwanese perspectives, policies, and issues.
// 2. Read and Analyze the Article
//     - Carefully read and understand the article’s main topic, purpose, and context.
//     - Identify recurring terms, central ideas, and emphasized points.
// 3. Categorize keyword:
//     - Extract and organize keywords into three distinct categories:
//       1) Themes: Overarching topics or main ideas (e.g., "能源政策").
//       2) Entities: Names of people, organizations, locations, or proper nouns (e.g., "台積電", "蔡英文").
//       3) Actions/Concepts: Important verbs, policies, or concepts central to the message (e.g., "推動綠能", "經濟改革").
// 4. Filter and Prioritize:
//     - Avoid generic terms or overused words that do not add value (e.g., "以及," "的").
//     - Ensure keywords are unique and meaningful, directly tied to the content.
//     - Avoid Simplified Chinese terms—focus exclusively on Traditional Chinese.

// Checklist
//   - At most 15–20 keywords or key phrases.
//   - Confirm that all keywords are in Traditional Chinese.
//   - Ensure extracted keywords reflect the article’s central ideas and regional context.

// Input Format:
// {
//     "article": "以繁體中文撰寫的文章"
// }

// Input Example:
// {
//     "article": "台灣政府正積極推動綠能轉型，目標於2050年達成淨零排放。同時，半導體產業的發展仍然是全球競爭的焦點。台積電持續領先，美國與歐盟也相繼投入資源建立自己的晶片供應鏈。"
// }

// Output Format:
// {
//     "themes": ["主題1", "主題2"],
//     "entities": ["實體1", "實體2", "實體3"],
//     "actions_concepts": ["行動1", "概念2", "政策3"]
// }

// Output Example:
// {
//     "themes": ["綠能轉型", "淨零排放", "半導體產業"],
//     "entities": ["台灣政府", "台積電", "美國", "歐盟"],
//     "actions_concepts": ["推動綠能轉型", "建立晶片供應鏈", "領先"]
// }`
// 	content := "以色列6月13日（以下皆指台灣時間）對伊朗發動大規模攻勢，聲稱要阻止伊朗發展核武，伊朗也隨之反擊，雙方激烈交火；6月22日，美國軍事介入以伊衝突，轟炸伊朗三個核設施，喊話伊朗必須立即停火。以色列和伊朗6月24日接受美國總統川普提出的停火方案，以終結他們這場歷時12天、撼動中東局勢的戰爭，不過川普說，以色列和伊朗都違反了停火協議。"
// 	payload, err := json.Marshal(map[string]string{
// 		"article": content,
// 	})
// 	require.NoError(t, err, "Failed to marshal payload to JSON")

// 	// model := "TwinkleAI/Llama-3.2-3B-F1-Resoning-Instruct:latest"
// 	model := "gemma3:latest"
// 	ollama := llm.NewDefaultOllamaClient()
// 	require.NotNil(t, ollama, "Ollama client should not be nil")
// 	messages := []api.Message{
// 		{
// 			Role:    string(llm.RoleSystem),
// 			Content: prompt,
// 		},
// 		{
// 			Role:    string(llm.RoleUser),
// 			Content: string(payload),
// 		},
// 	}
// 	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
// 	defer cancel()
// 	resp, err := ollama.Chat(ctx, model, messages, false, nil, nil)
// 	require.NoError(t, err, "Chatting with Ollama should not return an error")
// 	require.NotNil(t, resp, "Chat response should not be nil")
// 	require.NotEmpty(t, resp.Content, "Chat response content should not be empty")
// 	t.Log("Chat response content:", resp.Content)
// 	t.Log("Chat response thinking:", resp.Thinking)
// }
