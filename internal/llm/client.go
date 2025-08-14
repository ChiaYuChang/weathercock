package llm

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ChiaYuChang/weathercock/internal/global"
)

var (
	ErrModelNotFound          = errors.New("model not found")
	ErrModelHasBeenRegistered = errors.New("model has already been registered")
)

type BaseClient struct {
	Models            map[string]Model
	DefaultModels     map[ModelType]string
	SystemInstruction map[ModelType]string
}

func NewClient() *BaseClient {
	return &BaseClient{
		Models:        make(map[string]Model),
		DefaultModels: make(map[ModelType]string),
	}
}

func (cli *BaseClient) SetDefaultModel(t ModelType, name string) error {
	if _, ok := cli.Models[name]; !ok {
		return fmt.Errorf("%w: %s", ErrModelNotFound, name)
	}
	cli.DefaultModels[t] = name
	return nil
}

func (cli *BaseClient) DefaultModel(t ModelType) (Model, bool) {
	name, ok := cli.DefaultModels[t]
	if !ok {
		return nil, false
	}
	m, ok := cli.Models[name]
	return m, ok
}

func (cli BaseClient) HasModel(name string) bool {
	_, ok := cli.Models[name]
	return ok
}

func (cli *BaseClient) AddModel(model Model) {
	if _, exists := cli.Models[model.Name()]; exists {
		global.Logger.Warn().
			Str("model", model.Name()).
			Msg("model already exists, overwriting")
	}
	cli.Models[model.Name()] = model
}

func (cli *BaseClient) ListModels() []Model {
	models := make([]Model, 0, len(cli.Models))
	for _, model := range cli.Models {
		models = append(models, model)
	}

	slices.SortFunc(models, func(a, b Model) int {
		if a.Name() < b.Name() {
			return -1
		}
		return 1
	})

	return models
}

func (cli *BaseClient) WithModel(models ...Model) error {
	for _, model := range models {
		if _, ok := cli.Models[model.Name()]; ok {
			return fmt.Errorf("%w: %s", ErrModelHasBeenRegistered, model.Name())
		}
		cli.Models[model.Name()] = model
	}
	return nil
}

func (cli *BaseClient) WithSystemInstruction(t ModelType, instruction string) error {
	if !t.Valid() {
		return fmt.Errorf("%w: %s", ErrInvalidModelType, t)
	}
	cli.SystemInstruction[t] = instruction
	return nil
}
