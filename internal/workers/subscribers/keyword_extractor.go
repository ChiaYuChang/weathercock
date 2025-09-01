// Package subscribers contains implementations of NATS message handlers (workers).
package subscribers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/ChiaYuChang/weathercock/internal/workers"
	"github.com/ChiaYuChang/weathercock/internal/workers/publishers"
	"github.com/invopop/jsonschema"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// NATS stream, durable consumer, subject, and source names for the KeywordExtractorWorker.
const (
	KeywordExtractorWorkerStreamName  = "TASK"
	KeywordExtractorWorkerDurableName = "keyword-extractor-worker"
	KeywordExtractorWorkerSubject     = workers.TaskExtractKeywords
	KeywordExtractorWorkerSource      = "keyword-extractor-worker"
)

// Constants for OpenTelemetry span names, used for tracing.
const (
	KeywordExtractorSpanReadDataFromCache = "keyword-extractor.read-article-from-cache"
	KeywordExtractorSpanReadDataFromDB    = "keyword-extractor.read-article-from-db"
	KeywordExtractorSpanGenerateKeywords  = "keyword-extractor.generate-keywords"
	KeywordExtractorSpanInsertKeywords    = "keyword-extractor.insert-keywords-to-cache"
)

// Constants for retry logic when interacting with external services (e.g., LLM API).
const (
	MaxRetryTimes    = 3
	MinRetryInterval = 500 * time.Millisecond
	MaxRetryInterval = 10 * time.Second
)

// LLMCli is a helper struct to bundle an LLM client with its specific
// configuration (model, prompt) for this worker.
type LLMCli struct {
	client llm.LLM
	prompt string
	model  string
	config any
}

// NewLLM creates a new LLM client configuration.
func NewLLM(client llm.LLM, model, prompt string, config any) *LLMCli {
	return &LLMCli{
		client: client,
		prompt: prompt,
		model:  model,
		config: config,
	}
}

// KeywordExtractorOutput defines the expected JSON structure from the LLM.
// This is used with jsonschema to enforce a reliable output format.
type KeywordExtractorOutput struct {
	Keywords struct {
		Themes   []string `json:"themes"`
		Events   []string `json:"events"`
		Entities []string `json:"entities"`
		Actions  []string `json:"actions"`
	} `json:"keywords"`
	Relations []struct {
		Entity1  string `json:"entity1"`
		Entity2  string `json:"entity2"`
		Relation string `json:"relation"`
	} `json:"relations"`
}

// Flatten transforms the structured keywords into a flat slice of strings,
// suitable for simple storage or processing. Each keyword is prefixed with its type.
func (k KeywordExtractorOutput) Flatten() []string {
	keywords := make([]string, 0, len(k.Keywords.Themes)+len(k.Keywords.Events)+
		len(k.Keywords.Entities)+len(k.Keywords.Actions))
	for _, theme := range k.Keywords.Themes {
		keywords = append(keywords, fmt.Sprintf("theme:%s", theme))
	}

	for _, event := range k.Keywords.Events {
		keywords = append(keywords, fmt.Sprintf("event:%s", event))
	}

	for _, entity := range k.Keywords.Entities {
		keywords = append(keywords, fmt.Sprintf("entity:%s", entity))
	}

	for _, action := range k.Keywords.Actions {
		keywords = append(keywords, fmt.Sprintf("action:%s", action))
	}

	return keywords
}

// KeywordExtractorWorker is the main worker struct, holding all necessary dependencies
// like database connections, cache clients, and the LLM client.
type KeywordExtractorWorker struct {
	workers.BaseWorker
	storage   *storage.Storage
	valkey    *redis.Client
	llm       *LLMCli
	prompt    string
	publisher *publishers.Publisher
}

// NewKeywordExtractorWorker creates a new instance of the worker, initializing
// its base components and a dedicated publisher for sending completion events.
func NewKeywordExtractorWorker(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer,
	store *storage.Storage, valkey *redis.Client, llm *LLMCli) (*KeywordExtractorWorker, error) {
	baseWorker, err := workers.NewBaseWorker(nc, logger, tracer)
	if err != nil {
		return nil, err
	}

	pub := publishers.NewPublisher(
		fmt.Sprintf("%s-publisher", KeywordExtractorWorkerSource),
		baseWorker.JS, baseWorker.Logger, tracer)
	return &KeywordExtractorWorker{
		BaseWorker: *baseWorker,
		storage:    store,
		valkey:     valkey,
		llm:        llm,
		publisher:  pub,
	}, nil
}

func (w *KeywordExtractorWorker) Subject() string {
	return KeywordExtractorWorkerSubject
}

func (w *KeywordExtractorWorker) StreamName() string {
	return KeywordExtractorWorkerStreamName
}

func (w *KeywordExtractorWorker) DurableName() string {
	return KeywordExtractorWorkerDurableName
}

// ConsumerOptions defines the NATS consumer configuration.
func (w *KeywordExtractorWorker) ConsumerOptions() []nats.SubOpt {
	return []nats.SubOpt{
		nats.DeliverNew(),
		nats.AckExplicit(),
		nats.MaxAckPending(1),
		nats.ManualAck(),
	}
}

// log is a standardized logging helper to ensure consistent log formats for errors.
func (w KeywordExtractorWorker) log(cmd workers.CmdExtractKeywords,
	lvl zerolog.Level, msg string, start time.Time, err error, attrs map[string]any) {
	event := w.BaseWorker.Log(cmd.BaseMessage, lvl, start, attrs)
	event.Err(err).
		Int32("article_id", cmd.ArticleID)
	event.Msg(msg)
}

// Handle is the core logic for the worker. It processes a message from the NATS stream.
func (w *KeywordExtractorWorker) Handle(ctx context.Context, msg *nats.Msg) error {
	now := time.Now()
	w.Logger.Info().Msg("KeywordExtractorWorker received message")

	// 1. Parse and validate the incoming message.
	var cmd workers.CmdExtractKeywords
	var err error
	if err = json.Unmarshal(msg.Data, &cmd); err != nil {
		// If parsing fails, this is a permanent "poison pill" error.
		// We wrap it in ErrMalformedMessage to signal the runner to discard it.
		w.log(cmd, zerolog.ErrorLevel, "malformed message", now, err, map[string]any{
			"message": string(msg.Data),
		})
		return fmt.Errorf("%w: %s", workers.ErrMalformedMessage, err)
	}

	// 2. Get the article content, using a cache-then-database fallback strategy.
	// This ensures that if the cache is unavailable or stale, the worker can still
	// retrieve the necessary data from the primary database.
	var content string
	err = func(ctx context.Context) error {
		rCtx, rSpan := w.Tracer.Start(ctx, KeywordExtractorSpanReadDataFromCache)
		defer rSpan.End()

		// First, attempt to get the article content from the cache.
		var cErr error
		content, cErr = w.valkey.Get(rCtx, cmd.CacheKey).Result()
		if cErr != nil {
			// Fall back to the DB if error
			eMsg := "failed to read article from cache"
			if cErr == redis.Nil {
				eMsg = "cache missing"
			}
			w.log(cmd, zerolog.WarnLevel, eMsg, now, cErr, nil)
		}

		if cErr == nil && content == "" {
			w.log(cmd, zerolog.ErrorLevel, "empty content from cache", now, cErr, nil)
		}

		// If content is still empty (due to cache miss, error, or empty value), fetch from the database.
		if content == "" {
			article, dbErr := w.storage.UserArticles().GetByID(rCtx, cmd.ArticleID)
			if dbErr != nil {
				rSpan.RecordError(dbErr)
				w.log(cmd, zerolog.ErrorLevel, "failed to read article from db", now, dbErr, nil)
				return dbErr
			}
			content = article.Content
		}
		return nil
	}(ctx)
	if err != nil {
		// If we still have an error after the fallback, the task cannot proceed.
		w.log(cmd, zerolog.ErrorLevel, "failed to read article", now, err, nil)
		return fmt.Errorf("failed to read article: %w", err)
	}

	// 3. Generate keywords using the LLM client.
	// This step includes a robust retry mechanism with exponential backoff
	// to handle transient network issues or API rate limits when calling the LLM.
	var keywords KeywordExtractorOutput
	err = func(lCtx context.Context) error {
		schema := jsonschema.Reflect(KeywordExtractorOutput{})
		lCtx, lSpan := w.Tracer.Start(lCtx, KeywordExtractorSpanGenerateKeywords)
		defer lSpan.End()

		var resp *llm.GenerateResponse
		retry := 0
		// Retry loop with exponential backoff to handle transient LLM API failures.
		for err = nil; retry < MaxRetryTimes; retry++ {
			resp, err = w.llm.client.Generate(lCtx, &llm.GenerateRequest{
				Messages: []llm.Message{
					{
						Role:    llm.RoleSystem,
						Content: []string{w.prompt},
					},
					{
						Role:    llm.RoleUser,
						Content: []string{content},
					},
				},
				ModelName: w.llm.model,
				Schema: &llm.ResponseSchema{
					Name:        "keywords",
					Description: "keywords-extraction-results",
					S:           schema,
					Strict:      true,
				},
				Config: w.llm.config,
			})

			if err == nil {
				break // Success
			}
			time.Sleep(min(MaxRetryInterval, MinRetryInterval<<retry))
		}
		if err != nil {
			lSpan.RecordError(err)
			return fmt.Errorf("failed to generate keywords (3 retries): %w", err)
		}

		if err = json.Unmarshal([]byte(resp.Outputs[0]), &keywords); err != nil {
			lSpan.RecordError(err)
			return fmt.Errorf("failed to unmarshal keywords: %w", err)
		}
		return nil
	}(ctx)
	if err != nil {
		w.log(cmd, zerolog.ErrorLevel, "failed to generate keywords", now, err,
			map[string]any{
				"model":  w.llm.model,
				"prompt": w.prompt,
				"config": w.llm.config,
			})
		return err
	}

	// 4. Cache the results and publish a completion event.
	cachekey := fmt.Sprintf("%s.article.keywords", cmd.TaskID.String())
	vCtx, vSpan := w.Tracer.Start(ctx, KeywordExtractorSpanInsertKeywords)
	defer vSpan.End()
	err = w.valkey.Set(vCtx, cachekey, keywords, time.Hour*3).Err()
	if err != nil {
		vSpan.RecordError(err)
		w.log(cmd, zerolog.ErrorLevel, "failed to insert keywords to cache", now, err, nil)
		return fmt.Errorf("failed to insert keywords to cache: %w", err)
	}

	// 5. Publish an event to notify other services that keywords have been extracted.
	err = w.publisher.PublishNATSMessage(ctx, workers.KeywordsExtracted, workers.MsgKeywordsExtracted{
		BaseMessageWithElapsed: workers.BaseMessageWithElapsed{
			BaseMessage: workers.BaseMessage{
				TaskID:   cmd.TaskID,
				EventAt:  now.Unix(),
				Version:  workers.MessageVersion,
				CacheKey: cachekey,
			},
			ElapsedMs: time.Since(now).Milliseconds(),
		},
		ArticleID:      cmd.ArticleID,
		KeywordsCount:  len(keywords.Flatten()),
		RelationsCount: len(keywords.Relations),
	})
	if err != nil {
		w.log(cmd, zerolog.ErrorLevel, "failed to publish keywords", now, err, map[string]any{
			"cache_key": cachekey,
			"keywords":  keywords,
		})
		return fmt.Errorf("failed to publish keywords: %w", err)
	}
	w.log(cmd, zerolog.InfoLevel, "keywords extracted and published", now, nil, map[string]any{
		"cache_key": cachekey,
		"keywords":  keywords,
	})
	return nil
}
