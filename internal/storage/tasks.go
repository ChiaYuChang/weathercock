package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/google/uuid"
)

type Tasks Storage

func (s Storage) Task() Tasks {
	return Tasks(s)
}

func (t Tasks) CreateFromURL(ctx context.Context, url string) (uuid.UUID, error) {
	uid, err := t.db.CreateTask(ctx, models.CreateTaskParams{
		Source:        models.SourceTypeUrl,
		OriginalInput: url,
	})

	return uid, handlePgxErr(err)
}

func (t Tasks) CreateFromText(ctx context.Context, text string) (uuid.UUID, error) {
	uid, err := t.db.CreateTask(ctx, models.CreateTaskParams{
		Source:        models.SourceTypeText,
		OriginalInput: text,
	})
	return uid, handlePgxErr(err)
}

func (s Storage) UserTasks() UserTasks {
	return UserTasks{
		db: s.db,
	}
}

type UserTasks struct {
	db models.Querier
}

func (s UserTasks) Insert(ctx context.Context, source, input string, createdAt time.Time) (uuid.UUID, error) {
	var src models.SourceType
	err := src.Scan(source)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("unknown source type: %w", err)
	}
	// fmt.Println("Inserting task with source:", src, "and input:", input)

	uid, err := s.db.CreateTask(ctx, models.CreateTaskParams{
		Source:        src,
		OriginalInput: input,
	})

	if err != nil {
		fmt.Println("error after inserting task:", err)
		return uuid.UUID{}, handlePgxErr(err)
	}
	// fmt.Println("Task inserted successfully with ID:", uid)
	return uid, nil
}

func (s UserTasks) SelectByID(ctx context.Context, taskID uuid.UUID) (string, time.Time, error) {
	panic("not implemented")
}
