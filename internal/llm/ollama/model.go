package ollama

import (
	"github.com/ChiaYuChang/weathercock/internal/llm"
)

type OllamaModel struct {
	llm.BaseModel
	License      string         `json:"license"`
	ModelInfo    map[string]any `json:"model_info"`
	Capabilities []string       `json:"capabilities"`
}

// NewOllamaModel creates a new OllamaModel with the specified model type and name.
func NewOllamaModel(modelType llm.ModelType, name string) OllamaModel {
	return OllamaModel{
		BaseModel: llm.NewBaseModel(modelType, name),
	}
}
