package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

type OpenAI struct {
	client openai.Client
}

func NewOpenAI(opts ...option.RequestOption) *OpenAI {
	client := openai.NewClient(opts...)
	return &OpenAI{client: client}
}

type OpenAIResponse OpenAI

func (cli OpenAIResponse) New(ctx context.Context, model string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	input := responses.ResponseInputParam{}

	resp, err := cli.client.Responses.New(ctx, responses.ResponseNewParams{
		Background:      openai.Bool(false),
		Instructions:    openai.String(strings.Join(req.SystemInstruction, "\n")),
		MaxOutputTokens: openai.Int(int64(req.MaxOutputTokens)),
		Store:           openai.Bool(false),
		Temperature:     openai.Float(req.Temperature),
		TopLogprobs:     openai.Int(int64(req.TopLogProbs)),
		TopP:            openai.Float(req.TopP),
		Input:           responses.ResponseNewParamsInputUnion{OfInputItemList: input},
		Model:           shared.ChatModel(model),
	})

	if err != nil {
		return nil, err
	}

	if resp.Error.Code != "" {
		return nil, fmt.Errorf("OpenAI API error: %s - %s", resp.Error.Code, resp.Error.Message)
	}

	if resp.IncompleteDetails.Reason != "" {
		return nil, fmt.Errorf("OpenAI API incomplete response: %s", resp.IncompleteDetails.Reason)
	}

	msgs := []ChatCompletionMessage{}
	thinking := []string{}
	for _, output := range resp.Output {
		switch output.Type {
		case "message":
			msg := BaseMessage{
				role:    Role(output.Role),
				content: make([]InputText, len(output.Content)),
				Name:    output.Name,
			}

			for _, c := range output.Content {
				if c.Type == "text" {
					msg.content = append(msg.content, InputText{Text: c.Text})
				}
			}

			if len(output.Content) > 0 {
				msgs = append(msgs, msg)
			}
		case "reasoning":
			for _, summary := range output.Summary {
				thinking = append(thinking, summary.Text)
			}
		}
	}

	return &ChatCompletionResponse{
		ID:       resp.ID,
		Model:    model,
		Messages: msgs,
		Thinking: strings.Join(thinking, "\n"),
	}, nil
}

// implement ChatCompletion interface
type OpenAIChatCompletion OpenAI

func (cli OpenAIChatCompletion) New(ctx context.Context, model string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	msgs := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, msg := range req.Messages {
		switch m := msg.(type) {
		case BaseMessage:
			switch m.role {
			case RoleUser:
				var content openai.ChatCompletionUserMessageParamContentUnion
				if n := len(m.content); n == 1 {
					content = openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(m.content[0].Text),
					}
				} else {
					arr := make([]openai.ChatCompletionContentPartUnionParam, n)
					for i, c := range m.content {
						arr[i] = openai.ChatCompletionContentPartUnionParam{
							OfText: &openai.ChatCompletionContentPartTextParam{
								Text: c.Text,
								Type: "text",
							},
						}
					}
					content = openai.ChatCompletionUserMessageParamContentUnion{
						OfArrayOfContentParts: arr,
					}
				}

				msgs[i] = openai.ChatCompletionMessageParamUnion{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Name:    openai.String(m.Name),
						Content: content,
					},
				}
			case RoleSystem:
				var content openai.ChatCompletionSystemMessageParamContentUnion
				if n := len(m.content); n == 1 {
					content = openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(m.content[0].Text),
					}
				} else {
					arr := make([]openai.ChatCompletionContentPartTextParam, n)
					for i, c := range m.content {
						arr[i] = openai.ChatCompletionContentPartTextParam{
							Text: c.Text,
							Type: "text",
						}
					}
					content = openai.ChatCompletionSystemMessageParamContentUnion{
						OfArrayOfContentParts: arr,
					}
				}
				msgs[i] = openai.ChatCompletionMessageParamUnion{
					OfSystem: &openai.ChatCompletionSystemMessageParam{
						Name:    openai.String(m.Name),
						Content: content,
					},
				}
			}
		case ToolMessage:
		}
	}

	// 	cli.client.Chat.Completions.New(
	// 		ctx, openai.ChatCompletionNewParams{
	// 			Model: req.Model,
	// 		}
	// 	)
	return nil, nil
}
