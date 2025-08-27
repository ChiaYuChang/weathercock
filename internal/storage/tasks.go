package storage

import (
	"context"

	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/google/uuid"
)

type Tasks struct {
	Storage
}

func (s Storage) Task() Tasks {
	return Tasks{s}
}

func (t Tasks) InsertFromURL(ctx context.Context, url string,
	fn func(ctx context.Context, taskID uuid.UUID) error) (uuid.UUID, error) {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return uuid.UUID{}, handlePgxErr(err)
	}
	defer tx.Rollback(ctx)

	var uid uuid.UUID
	if uid, err = t.Queries.WithTx(tx).InsertUserTask(ctx, models.InsertUserTaskParams{
		Source:        models.SourceTypeUrl,
		OriginalInput: url,
	}); err != nil {
		return uuid.UUID{}, handlePgxErr(err)
	}

	if fn != nil {
		if err = fn(ctx, uid); err != nil {
			return uuid.UUID{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return uuid.UUID{}, handlePgxErr(err)
	}
	return uid, nil
}

func (t Tasks) InsertFromText(ctx context.Context, text string,
	fn func(ctx context.Context, taskID uuid.UUID) error) (uuid.UUID, error) {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return uuid.UUID{}, handlePgxErr(err)
	}
	defer tx.Rollback(ctx)

	var uid uuid.UUID
	if uid, err = t.Queries.WithTx(tx).InsertUserTask(ctx, models.InsertUserTaskParams{
		Source:        models.SourceTypeText,
		OriginalInput: text,
	}); err != nil {
		return uuid.UUID{}, handlePgxErr(err)
	}

	if fn != nil {
		if err = fn(ctx, uid); err != nil {
			return uuid.UUID{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return uuid.UUID{}, handlePgxErr(err)
	}
	return uid, nil
}
