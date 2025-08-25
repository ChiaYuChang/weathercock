package workers

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/google/uuid"
)

// Publish while event has occurred
const (
	// task has been created
	TaskCreated = "task.created"
	// news article has been scraped
	ArticleScraped = "article.scraped"
	// keywords for the article have been extracted
	KeywordsExtracted = "article.keywords.extracted"
	// embedding for the article has been created
	EmbeddingCreated = "article.embedding.created"
)

// Publish while event needs to be performed
const (
	// scrape a news article
	TaskScrape = "task.scrape"
	// generate a title for the article
	TaskGenerateTitle = "task.generate_title"
	// extract keywords from the article
	TaskExtractKeywords = "task.extract.keyword"
	// create an embedding for the article
	TaskCreateEmbedding = "task.create.embedding"
	// update the status of the task
	TaskUpdateStatus = "task.update.status"
	// log the task
	TaskLog = "task.logs"
)

var (
	ErrInvalidEmbedType = errors.New("invalid embed type, must be query or passage")
	ErrInvalidLogLevel  = errors.New("invalid log level")
)

type EmbedType string

const (
	EmbedTypeQuery   EmbedType = "query"
	EmbedTypePassage EmbedType = "passage"
)

func (e EmbedType) String() string {
	return string(e)
}

func (e *EmbedType) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		*e = EmbedType(string(v))
	case string:
		*e = EmbedType(v)
	default:
		return fmt.Errorf("unsupported type for EmbedType: %T", src)
	}
	return nil
}

func (e EmbedType) Valid() bool {
	switch e {
	case EmbedTypeQuery, EmbedTypePassage:
		return true
	}
	return false
}

func (e EmbedType) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

func (e *EmbedType) UnmarshalJSON(data []byte) error {
	var embedType string
	if err := json.Unmarshal(data, &embedType); err != nil {
		return err
	}

	if err := e.Scan(embedType); err != nil {
		return err
	}

	if !e.Valid() {
		return ErrInvalidEmbedType
	}
	return nil
}

// LogLevel defines log levels.
type LogLevel string

const (
	DebugLogLevel LogLevel = "Debug"
	InfoLogLevel  LogLevel = "Info"
	WarnLogLevel  LogLevel = "Warn"
	ErrorLogLevel LogLevel = "Error"
	FatalLogLevel LogLevel = "Fatal"
	PanicLogLevel LogLevel = "Panic"
	TraceLogLevel LogLevel = "Trace"
)

func (l LogLevel) String() string {
	return string(l)
}

func (l *LogLevel) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		*l = LogLevel(string(v))
	case string:
		*l = LogLevel(v)
	default:
		return fmt.Errorf("unsupported type for Level: %T", src)
	}
	return nil
}

func (l LogLevel) Valid() bool {
	switch l {
	case
		DebugLogLevel,
		InfoLogLevel,
		WarnLogLevel,
		ErrorLogLevel,
		FatalLogLevel,
		PanicLogLevel,
		TraceLogLevel:
		return true
	}
	return false
}

func (l LogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

func (l *LogLevel) UnmarshalJSON(data []byte) error {
	var level string
	if err := json.Unmarshal(data, &level); err != nil {
		return err
	}

	if err := l.Scan(level); err != nil {
		return err
	}

	if !l.Valid() {
		return ErrInvalidLogLevel
	}
	return nil
}

// BaseMessage is the base struct for all messages.
// Timestamp is in Unix format.
type BaseMessage struct {
	TaskID   uuid.UUID `json:"task_id"`
	UserID   int32     `json:"user_id,omitempty"`
	CacheKey string    `json:"cache_key,omitempty"`
	EventAt  int64     `json:"event_at"`
}

func (msg BaseMessage) Time() time.Time {
	return time.Unix(msg.EventAt, 0)
}

type BaseMessageWithElapsed struct {
	BaseMessage
	ElapsedMs int64 `json:"elapsed_ms,omitempty"`
}

type MsgTaskCreated struct {
	BaseMessageWithElapsed
	Source        models.SourceType `json:"source"`
	OriginalInput string            `json:"original_input,omitempty"`
}

type MsgArticleScraped struct {
	BaseMessageWithElapsed
	ArticleID int32 `json:"article_id"`
}

type MsgKeywordsExtracted struct {
	BaseMessageWithElapsed
	ArticleID     int32 `json:"article_id"`
	KeywordsCount int   `json:"keywords_count"`
}

type MsgEmbeddingCreated struct {
	BaseMessageWithElapsed
	ArticleID int32 `json:"article_id"`
}

type CmdScrapeArticle struct {
	BaseMessage
	URL string `json:"url,omitempty"`
}

type CmdGenerateTitle struct {
	BaseMessage
	Content string `json:"content"` // Added Content field
}

type CmdExtractKeywords struct {
	BaseMessage
	ArticleID int32 `json:"article_id"`
}

type CmdCreateEmbedding struct {
	BaseMessage
	ArticleID int32     `json:"article_id"`
	EmbedType EmbedType `json:"embed_type"`
}

type CmdUpdateTaskStatus struct {
	BaseMessage
	Status models.TaskStatus `json:"status"`
}

type CmdTaskLog struct {
	BaseMessage
	Level   LogLevel `json:"level"`
	Message string   `json:"message"`
}