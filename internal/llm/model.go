package llm

import (
	"encoding/json"
	"fmt"
)

type ModelType string

const (
	ModelGenerate ModelType = "generate"
	ModelEmbed    ModelType = "embedding"
)

var (
	ErrInvalidModelType = fmt.Errorf("invalid model type")
)

var ModelTypeList = []ModelType{ModelGenerate, ModelEmbed}

func (m ModelType) String() string {
	return string(m)
}

func (m ModelType) Valid() bool {
	for _, valid := range ModelTypeList {
		if m == valid {
			return true
		}
	}
	return false
}

func (m ModelType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(m))
}

func (m *ModelType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*m = ModelType(s)
	if !m.Valid() {
		return fmt.Errorf("%w: %s", ErrInvalidModelType, s)
	}
	return nil
}

type Model interface {
	Type() ModelType
	Name() string
}

type BaseModel struct {
	modelType ModelType
	name      string
}

func NewBaseModel(modelType ModelType, name string) BaseModel {
	return BaseModel{
		modelType: modelType,
		name:      name,
	}
}

func (m BaseModel) Type() ModelType { return m.modelType }
func (m BaseModel) Name() string    { return m.name }

func (m BaseModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type ModelType `json:"type"`
		Name string    `json:"name"`
	}{
		Type: m.modelType,
		Name: m.name,
	})
}

func (m *BaseModel) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Type ModelType `json:"type"`
		Name string    `json:"name"`
	}
	var ali Alias
	if err := json.Unmarshal(data, &ali); err != nil {
		return err
	}

	m.modelType = ModelType(ali.Type)
	if !m.modelType.Valid() {
		return fmt.Errorf("%w: %s", ErrInvalidModelType, ali.Type)
	}
	m.name = ali.Name
	return nil
}
