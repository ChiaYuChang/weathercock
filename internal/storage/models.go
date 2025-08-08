package storage

import (
	"context"

	"github.com/ChiaYuChang/weathercock/internal/models"
)

func (s Storage) Models() Models {
	return Models{
		db: s.db,
	}
}

type Models struct {
	db models.Querier
}

// Insert adds a new LLM model to the database and returns its ID.
func (m Models) Insert(ctx context.Context, name string) (int32, error) {
	mID, err := m.db.InsertModel(ctx, name)
	if err != nil {
		return 0, handlePgxErr(err)
	}
	return mID, nil
}

// GetByID retrieves the LLM model by its ID.
func (m Models) GetByID(ctx context.Context, id int32) (models.Model, error) {
	model, err := m.db.GetModelByID(ctx, id)
	if err != nil {
		return models.Model{}, handlePgxErr(err)
	}

	return models.Model{
		ID:   model.ID,
		Name: model.Name,
	}, nil

}

// GetByName retrieves a model by its name.
func (m Models) GetByName(ctx context.Context, name string) (models.Model, error) {
	model, err := m.db.GetModelByName(ctx, name)
	if err != nil {
		return models.Model{}, handlePgxErr(err)
	}

	return models.Model{
		ID:   model.ID,
		Name: model.Name,
	}, nil
}

// List retrieves a list of models with pagination support.
func (m Models) List(ctx context.Context, limit, offset int32) ([]models.Model, error) {
	rows, err := m.db.ListModels(ctx, models.ListModelsParams{
		Limit:  limit,
		Offset: offset,
	})

	if err != nil {
		return nil, handlePgxErr(err)
	}

	result := make([]models.Model, len(rows))
	for i, model := range rows {
		result[i] = models.Model{
			ID:   model.ID,
			Name: model.Name,
		}
	}

	return result, nil
}

// DeleteByID removes a model by its ID.
func (m Models) DeleteByID(ctx context.Context, id int32) error {
	err := m.db.DeleteModelByID(ctx, id)
	if err != nil {
		return handlePgxErr(err)
	}
	return nil
}
