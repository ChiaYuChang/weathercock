package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
)

const (
	EmbedStateOk        = ""
	EmbedStateTruncated = "truncated"
	EmbedStateError     = "error"
)

var (
	ErrContextShouldNotBeNull = errors.New("context in request should not be null")
	ErrRequestShouldNotBeNull = errors.New("request should not be null")
	ErrNoInput                = errors.New("no input provided in request")
)

type Role string

type Text struct {
	Content string
	Prefix  string
}

func NewSimpleText(content string) Text {
	return Text{Content: content}
}

func (t Text) String() string {
	return fmt.Sprintf("%s%s", t.Prefix, t.Content)
}

type EmbedInput interface {
	String() string
}

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content []string
}

type GenerateRequest struct {
	context.Context
	Messages  []Message
	ModelName string
	Config    any
}

type GenerateResponse struct {
	Outputs []string
	Raw     any
}

type EmbedRequest struct {
	Ctx       context.Context
	Inputs    []EmbedInput
	ModelName string
	Config    any
}

type EmbedResponse struct {
	Model      string      `json:"model,omitempty"`
	Embeddings []Embedding `json:"embeddings,omitempty"`
	Raw        any         `json:"raw,omitempty"`
}

type Embedding struct {
	State  string    `json:"state,omitempty"`
	Values []float32 `json:"values,omitempty"`
}

func (embed Embedding) Dim() int {
	return len(embed.Values)
}

func (embed Embedding) String() string {
	vals := embed.Values[:min(10, len(embed.Values))]
	strs := make([]string, len(vals))
	for i, v := range vals {
		strs[i] = fmt.Sprintf("%6.3f", v)
	}

	state := utils.IfElse(embed.State == "", "ok", embed.State)
	return fmt.Sprintf("{state: %s, dim: %d, value: %s ...}",
		state, len(embed.Values), strings.Join(strs, ", "))
}

type BatchRequest struct {
	Ctx          context.Context    `json:"-"`
	ModelName    string             `json:"model_name"`
	BatchJobName string             `json:"batch_job_name"`
	Requests     []*GenerateRequest `json:"requests"`
	Config       any                `json:"config"`
}

type BatchResponse struct {
	ID        string              `json:"id"`
	Status    string              `json:"status"`
	IsDone    bool                `json:"is_done"`
	CreatedAt time.Time           `json:"created_at"`
	StartAt   time.Time           `json:"start_at"`
	EndAt     time.Time           `json:"end_at"`
	UpdateAt  time.Time           `json:"update_at"`
	Responses []*GenerateResponse `json:"responses"`
	Raw       any                 `json:"raw"`
}

type BatchRetrieveRequest struct {
	Ctx    context.Context `json:"-"`
	ID     string          `json:"id"`
	Config any             `json:"config"`
}

type BatchCancelRequest struct {
	Ctx    context.Context `json:"-"`
	ID     string          `json:"id"`
	Config any             `json:"config"`
}
