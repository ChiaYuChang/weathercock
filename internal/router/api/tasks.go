package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/ChiaYuChang/weathercock/internal/workers"
	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.ErrBadRequest

// UserTasks provides methods to manage UserTasks in the repository.
type UserTasks struct {
	*Repo
	*validator.Validate
}

// Convert Repo to Tasks interface which provides methods for task management.
func (r *Repo) UserTask(validator *validator.Validate) TaskEndpoint {
	return UserTasks{
		Repo:     r,
		Validate: validator,
	}
}

func (t UserTasks) InsertFromText(r *http.Request) (taskID uuid.UUID, err error) {
	if err = r.ParseForm(); err != nil {
		e := errors.ErrBadRequest.Clone()
		e.Details = append(e.Details, "failed to parse form data")
		e.Warp(err)
		return uuid.Nil, e
	}

	qURL := r.Form["query_url"][0]
	vCtx, vCancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer vCancel()
	err = t.Validate.VarCtx(vCtx, qURL, "url,required")
	if err != nil {
		e := errors.ErrBadRequest.Clone()
		e.Details = append(e.Details, "invalid URL format")
		e.Warp(err)
		return uuid.Nil, e
	}

	if u, err := url.Parse(qURL); err != nil || u.Hostname() != "tw.news.yahoo.com" {
		e := errors.ErrBadRequest.Clone()
		e.Details = append(e.Details, "only support Yahoo news URL (tw.news.yahoo.com)")
		return uuid.Nil, e
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	taskID, err = t.Storage.Task().InsertFromURL(ctx, qURL, func(ctx context.Context, taskID uuid.UUID) error {
		payload, err := json.Marshal(workers.CmdScrapeArticle{
			BaseMessage: workers.BaseMessage{TaskID: taskID},
			URL:         qURL,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal scrape task payload: %w", err)
		}

		err = t.Publisher.PublishNATSMessage(ctx, workers.TaskScrape, payload)
		if err != nil {
			return fmt.Errorf("failed to publish scrape task: %w", err)
		}
		return nil

	})

	if err != nil {
		e := errors.ErrDBError.Clone()
		e.Details = append(e.Details, "failed to create task")
		e.Warp(err)
		return uuid.Nil, e
	}
	return taskID, nil
}

func (t UserTasks) InsertFromURL(r *http.Request) (taskID uuid.UUID, err error) {
	if err = r.ParseForm(); err != nil {
		e := errors.ErrBadRequest.Clone()
		e.Details = append(e.Details, "failed to parse form data")
		e.Warp(err)
		return uuid.Nil, e
	}

	rawText := r.Form["query_text"][0]
	text := strings.TrimSpace(string(rawText))
	if len(text) == 0 {
		e := errors.ErrNoContent.Clone().
			WithDetails("empty query text").
			Warp(err)
		return uuid.Nil, e
	}

	if found, p := llm.DetectLLMInjection(text); found {
		e := errors.ErrContentContainsMaliciousPrompt.Clone().
			WithDetails(fmt.Sprintf("potential malicious prompt: %s", p))
		return uuid.Nil, e
	}

	// Detect if the content contains titles (start with # at the first line)
	contents := strings.Split(text, "\n")
	var title string
	if len(contents) > 0 && strings.HasPrefix(contents[0], "#") {
		title = strings.TrimSpace(contents[0][1:]) // remove the leading '#'
		contents = contents[1:]                    // remove the title from the contents
	}

	// Trim and remove empty lines
	end := 0
	for i := 0; i < len(contents); i++ {
		contents[end] = strings.TrimSpace(contents[i])
		if len(contents[end]) > 0 {
			end++
		}
	}

	if end == 0 {
		e := errors.ErrNoContent.Clone().WithDetails("empty query text")
		return uuid.Nil, e
	}
	contents = contents[:end]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	taskID, err = t.Storage.Task().InsertFromText(ctx, text, func(ctx context.Context, taskID uuid.UUID) error {
		if len(title) == 0 {
			err = t.Publisher.PublishNATSMessage(ctx,
				workers.TaskGenerateTitle,
				workers.CmdGenerateTitle{
					BaseMessage: workers.BaseMessage{
						TaskID: taskID,
					},
					Content: text,
				})
			if err != nil {
				return fmt.Errorf("failed to publish generate title task: %w", err)
			}
		}

		pipe := t.Storage.Cache.Pipeline()
		const ttl = 60 * time.Minute
		pipe.Set(ctx, fmt.Sprintf("task.%s.title", taskID.String()), title, ttl)
		pipe.Set(ctx, fmt.Sprintf("task.%s.contents", taskID.String()), contents, ttl)
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("failed to execute cache pipeline: %w", err)
		}
		return nil
	})

	if err != nil {
		e := errors.ErrDBError.Clone().
			WithDetails("failed to create task").
			Warp(err)
		return uuid.Nil, e
	}
	return
}

func (t UserTasks) Get(r *http.Request) (*models.UsersTask, error) {
	taskID, err := uuid.Parse(r.PathValue("task_id"))
	if err != nil {
		e := errors.ErrBadRequest.Clone().
			WithDetails("invalid task_id format").
			Warp(err)
		return nil, e
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	task, err := t.Storage.Queries.GetUserTask(ctx, taskID)
	if err != nil {
		pge, ok := errors.NewPGErr(err)
		var e *errors.Error
		if ok {
			e = errors.FromPgError(pge)
		} else {
			e = errors.ErrDBError.Clone().
				Warp(err).
				WithDetails("failed to get task")
		}
		return nil, e
	}
	return &task, nil
}

func (t UserTasks) UpdateStatus(r *http.Request) error {
	taskID, err := uuid.Parse(r.PathValue("task_id"))
	if err != nil {
		e := errors.ErrBadRequest.Clone().
			WithDetails("invalid task_id format").
			Warp(err)
		return e
	}

	status := models.TaskStatus(r.URL.Query().Get("status"))
	if !status.Valid() {
		e := errors.ErrBadRequest.Clone()
		if status == "" {
			e.WithDetails("missing status query parameter: status")
			return e
		}
		e.WithDetails(fmt.Sprintf("invalid status: %s", status))
		return e
	}

	err = t.Repo.Storage.Queries.UpdateUserTaskStatus(
		context.Background(), models.UpdateUserTaskStatusParams{
			TaskStatus: status,
			TaskID:     taskID,
		})

	if err != nil {
		pge, ok := errors.NewPGErr(err)
		var e *errors.Error
		if ok {
			e = errors.FromPgError(pge)
		} else {
			e = errors.ErrDBError.Clone().
				WithDetails("failed to get task").
				Warp(err)
		}
		return e
	}
	return nil
}
