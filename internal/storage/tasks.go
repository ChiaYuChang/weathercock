package storage

import (
	"context"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/google/uuid"
)

type Tasks Storage

func (s Storage) Task() Tasks {
	return Tasks(s)
}

func (t Tasks) CreateFromURL(ctx context.Context, url string) (uuid.UUID, error) {
	uid, err := t.DB.CreateTask(ctx, models.CreateTaskParams{
		Source:        models.SourceTypeUrl,
		OriginalInput: url,
	})

	return uid, handlePgxErr(err)
}

func (t Tasks) CreateFromText(ctx context.Context, text string) (uuid.UUID, error) {
	uid, err := t.DB.CreateTask(ctx, models.CreateTaskParams{
		Source:        models.SourceTypeText,
		OriginalInput: text,
	})
	return uid, handlePgxErr(err)
}

type UserTasks struct {
	DB models.Querier
}

func (s UserTasks) Insert(ctx context.Context, taskID uuid.UUID, name string, createdAt time.Time) error {
	panic("not implemented")
}

func (s UserTasks) SelectByID(ctx context.Context, taskID uuid.UUID) (string, time.Time, error) {
	panic("not implemented")
}
