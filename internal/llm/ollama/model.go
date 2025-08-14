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

func NewOllamaModel(modelType llm.ModelType, name string) OllamaModel {
	return OllamaModel{
		BaseModel: llm.NewBaseModel(modelType, name),
	}
}

// func (model OllamaModel) MarshalJSON() ([]byte, error) {
// 	type Alias OllamaModel
// 	return json.Marshal(struct {
// 		Name string        `json:"name"`
// 		Type llm.ModelType `json:"type"`
// 		*Alias
// 	}{
// 		Name:  model.Name(),
// 		Type:  model.Type(),
// 		Alias: Alias(model),
// 	})
// }
