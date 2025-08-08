package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	ollama "github.com/ollama/ollama/api"
)

type Ollama struct {
	client *ollama.Client
}

func NewOllama(url *url.URL, cli *http.Client) *Ollama {
	return &Ollama{
		client: ollama.NewClient(url, cli),
	}
}

func (o *Ollama) Heartbeat(ctx context.Context) error {
	if err := o.client.Heartbeat(ctx); err != nil {
		return fmt.Errorf("ollama: heartbeat failed: %w", err)
	}
	return nil
}

type OllamaChatCompletion struct {
	client *ollama.Client
	opts   map[string]any
	sehema json.RawMessage
	Think  bool
}

func (o *OllamaChatCompletion) WithOptions(opts map[string]any) *OllamaChatCompletion {
	o.opts = opts
	return o
}

func (o *OllamaChatCompletion) WithSchema(schema json.RawMessage) *OllamaChatCompletion {
	o.sehema = schema
	return o
}

func (o *OllamaChatCompletion) New(ctx context.Context, model string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Convert ChatCompletionRequest to Ollama request format
	system := []string{}
	for _, c := range req.SystemInstruction {
		system = append(system, c)
	}

	content := []string{}
	for _, msg := range req.Messages {
		switch m := msg.(type) {
		case BaseMessage:
			switch m.role {
			case RoleSystem:
				for _, c := range m.content {
					system = append(system, c.Text)
				}
			case RoleUser:
				for _, c := range m.content {
					content = append(content, c.Text)
				}
			default:
				log.Printf("ollama: unsupported role %s in message, skipping", m.role)
			}
		default:
			log.Printf("ollama: unsupported role %s in message, skipping", m.Role())
		}
	}

	ollamaReq := &ollama.GenerateRequest{
		Model:   model,
		System:  strings.Join(system, "\n"),
		Prompt:  strings.Join(content, "\n"),
		Stream:  utils.Ptr(req.Stream),
		Raw:     true,
		Options: o.opts,
		Think:   utils.Ptr(o.Think),
	}

	resp := &ChatCompletionResponse{}
	err := o.client.Generate(ctx, ollamaReq, func(gr ollama.GenerateResponse) error {
		resp.ID = "ollama not supported"
		resp.Messages = append(resp.Messages,
			NewBaseMessage(RoleSystem, "", gr.Response))
		resp.Model = gr.Model
		resp.Thinking = gr.Thinking
		resp.resp = gr
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, nil
}
