package llm

import (
	"errors"
	"fmt"
	"io"
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
	ErrRequestShouldNotBeNull = errors.New("request should not be null")
	ErrNoInput                = errors.New("no input provided in request")
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Request interface {
	Endpoint() string
}

type Message struct {
	Role    Role
	Content []string
}

type GenerateRequest struct {
	Messages  []Message
	ModelName string
	Config    any
}

func (req GenerateRequest) Endpoint() string {
	return "generate"
}

type GenerateResponse struct {
	Outputs []string
	Raw     any
}

type EmbedRequest struct {
	Inputs    []EmbedInput
	ModelName string
	Config    any
}

func (req EmbedRequest) Endpoint() string {
	return "embedding"
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
	ModelName         string            `json:"model_name"`
	BatchJobName      string            `json:"batch_job_name"`
	Endpoint          string            `json:"endpoint"`
	Requests          []Request         `json:"requests"`
	Metadata          map[string]string `json:"meta_data"`
	ReadWriter        io.ReadWriter     `json:"read_writer"`
	FileUploadConfig  any               `json:"file_upload_config"`
	BatchCreateConfig any               `json:"batch_create_config"`
}

type BatchResponse struct {
	HTTPStatusCode int       `json:"http_status_code,omitempty"`
	HTTPMessage    string    `json:"http_message,omitempty"`
	ID             string    `json:"id"`
	OutputFileID   string    `json:"output_file_id"`
	InputFileID    string    `json:"input_file_id"`
	Status         string    `json:"status"`
	IsDone         bool      `json:"is_done"`
	CreatedAt      time.Time `json:"created_at"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	UpdateAt       time.Time `json:"update_at"`
	Responses      [][]byte  `json:"responses"`
	Raw            any       `json:"raw"`
}

type BatchRetrieveRequest struct {
	ID                string `json:"id"`
	StatusCheckConfig any    `json:"status_check_config"`
	RetrieveConfig    any    `json:"retrieve_config"`
}

type BatchCancelRequest struct {
	ID     string `json:"id"`
	Config any    `json:"config"`
}
