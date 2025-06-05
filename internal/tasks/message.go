package tasks

import (
	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/google/uuid"
)

const (
	Prefix          = "task."
	Create          = "task.create"
	ExtractKeywords = "task.extract_keywords"
	Embedding       = "task.embedding"
	Done            = "task.done"
	Failed          = "task.failed"
	Error           = "task.error"
)

type TaskMessage struct {
	TaskID uuid.UUID         `json:"task_id"`
	Source models.SourceType `json:"source"`
	Input  string            `json:"original_input"`
}
