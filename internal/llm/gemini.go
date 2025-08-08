package llm

import (
	"context"
	"fmt"
	"log"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"google.golang.org/genai"
)

type Gemini struct {
	client *genai.Client
}

func NewGemini(ctx context.Context, config *genai.ClientConfig) (*Gemini, error) {
	client, err := genai.NewClient(ctx, config)
	if err != nil {
		return nil, err
	}

	return &Gemini{client: client}, nil
}

func (g *Gemini) ChatCompletion(history ...*genai.Content) *GeminiChatCompletion {
	return &GeminiChatCompletion{client: g.client}
}

func (g *GeminiChatCompletion) WithTools(tools ...*genai.Tool) *GeminiChatCompletion {
	g.Tools = tools
	return g
}

func (g *GeminiChatCompletion) WithHistory(history ...*genai.Content) *GeminiChatCompletion {
	g.history = append(g.history, history...)
	return g
}

type GeminiChatCompletion struct {
	client  *genai.Client
	history []*genai.Content
	Tools   []*genai.Tool
}

// implement ChatCompletion interface
func (g *GeminiChatCompletion) New(ctx context.Context, model string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	system := &genai.Content{Role: "model"}
	for _, c := range req.SystemInstruction {
		system.Parts = append(system.Parts, &genai.Part{Text: c})
	}

	content := []*genai.Content{}
	for _, msg := range req.Messages {
		switch m := msg.(type) {
		case BaseMessage:
			switch m.role {
			case RoleSystem:
				for _, c := range m.content {
					system.Parts = append(system.Parts,
						&genai.Part{
							Text: c.Text,
						})
				}
			case RoleUser:
				for _, c := range m.content {
					content = append(content, &genai.Content{
						Parts: []*genai.Part{{Text: c.Text}},
						Role:  m.role.String(),
					})
				}
			default:
				log.Printf("Unsupported role: %s", m.role)
			}
		case ToolMessage:
			log.Println("ToolMessage not implemented yet, skipping")
		default:
			log.Printf("Unsupported message type: %T, skipping", msg)
		}
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction:  system,
		Temperature:        utils.Ptr(float32(req.Temperature)),
		TopP:               utils.Ptr(float32(req.TopP)),
		MaxOutputTokens:    int32(req.MaxOutputTokens),
		Seed:               utils.Ptr(req.Seed),
		ResponseJsonSchema: req.Schema,
		Tools:              g.Tools,
	}

	resp, err := g.client.Models.GenerateContent(ctx, model, content, config)
	if err != nil {
		return nil, err
	}

	msgs := []ChatCompletionMessage{}
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			msg := NewBaseMessage(
				Role(resp.Candidates[0].Content.Role),
				"",
				part.Text,
			)
			msgs = append(msgs, msg)
		}
	}

	return &ChatCompletionResponse{
		ID:       "Gemini do not provide ID",
		Model:    fmt.Sprintf("%s:%s", model, resp.ModelVersion),
		Messages: msgs,
		resp:     resp,
	}, nil
}
