package storage

import (
	"context"

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
