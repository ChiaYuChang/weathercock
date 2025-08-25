package api

import (
	"context"
	"fmt"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.ErrBadRequest

// Tasks provides methods to manage tasks in the repository.
type Tasks *Repo

// CreateTask creates a new task with the provided URL or text.
// It validates the input and checks for potential LLM injection patterns in
// the text.
func (r *Repo) Create(ctx context.Context, url, text string) (uuid.UUID, error) {
	if url == "" && text == "" {
		err := errors.ErrValidationFailed.Clone()
		err.Message = "Either URL or Text must be provided"
		return uuid.Nil, err
	}

	if url != "" {
		vCtx, vCancel := context.WithTimeout(ctx, 1*time.Second)
		defer vCancel()
		global.Validator().VarCtx(vCtx, url, "url")

		sCtv, sCancel := context.WithTimeout(ctx, 5*time.Second)
		defer sCancel()
		return r.Storage.Task().CreateFromURL(sCtv, url, nil)
	}

	vCtx, vCancel := context.WithTimeout(ctx, 1*time.Second)
	defer vCancel()
	global.Validator().VarCtx(vCtx, text, "min=10,max=3000")

	// Check for potential LLM injection patterns in the text
	if found, pattern := llm.DetectLLMInjection(text); found {
		err := errors.ErrBadRequest.Clone()
		err.Message = "Input text contains potential LLM injection patterns"
		err.Details = []string{
			"Please remove any suspicious content and try again.",
			"Do not try to manipulate the AI's behavior with special instructions.",
			fmt.Sprintf("Detected pattern: %s", pattern),
		}
		return uuid.Nil, err
	}

	sCtv, sCancel := context.WithTimeout(ctx, 5*time.Second)
	defer sCancel()
	return r.Storage.Task().CreateFromText(sCtv, text, nil)
}
