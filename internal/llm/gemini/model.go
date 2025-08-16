package gemini

import (
	"encoding/json"

	"github.com/ChiaYuChang/weathercock/internal/llm"
)

type GeminiModel struct {
	DesplayName      string   `json:"display_name"`
	Version          string   `json:"version"`
	Description      string   `json:"description"`
	InputTokenLimit  int32    `json:"input_token_limit"`
	OutputTokenLimit int32    `json:"output_token_limit"`
	SupportedActions []string `json:"supported_actions"`
	llm.BaseModel
}

// MarshalJSON customizes the JSON marshaling of GeminiModel.
func (model GeminiModel) MarshalJSON() ([]byte, error) {
	type Alias GeminiModel
	return json.Marshal(&struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Alias
	}{
		Name:  model.Name(),
		Type:  string(model.Type()),
		Alias: Alias(model),
	})
}
