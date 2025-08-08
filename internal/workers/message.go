package tasks

import (
	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/google/uuid"
)

const (
	Prefix = "task."
)

const (
	Create          = Prefix + "create"
	Scrape          = Prefix + "scrape"
	GenerateTitle   = Prefix + "generate_title"
	ExtractKeywords = Prefix + "extract_keywords"
	Embedding       = Prefix + "embedding"
	Done            = Prefix + "done"
	Failed          = Prefix + "failed"
	Error           = Prefix + "error"
	Log             = Prefix + "log"
)

type TaskMessage struct {
	TaskID uuid.UUID         `json:"task_id"`
	Source models.SourceType `json:"source"`
	Input  string            `json:"original_input"`
}

type ScrapeTaskPayload struct {
	TaskID uuid.UUID `json:"task_id"`
	URL    string    `json:"url"`
}
