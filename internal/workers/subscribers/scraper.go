// Package subscribers contains implementations of NATS message handlers (workers).
package subscribers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/scrapers"
	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/ChiaYuChang/weathercock/internal/workers"
	"github.com/ChiaYuChang/weathercock/internal/workers/publishers"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"

	"github.com/nats-io/nats.go"
)

// NATS stream, durable consumer, subject, and source names for the ScraperWorker.
const (
	ScraperWorkerStreamName  = "TASK"
	ScraperWorkerDurableName = "scraper-worker"
	ScraperWorkerSubject     = workers.TaskScrape
	ScraperWorkerSource      = "scraper-worker"
)

const (
	ScraperWorkerSpanFetch       = "scrape.article.fetch"
	ScraperWorkerSpanParse       = "scrape.article.parse"
	ScraperWorkerSpanInsertDB    = "scrape.article.insert-db"
	ScraperWorkerSpanInsertCache = "scrape.article.insert-cache"
)

// ScraperWorker implements the Handler interface for scraping articles from the web.
type ScraperWorker struct {
	workers.BaseWorker
	storage   *storage.Storage
	valkey    *redis.Client
	publisher *publishers.Publisher
	httpCli   *http.Client
	headers   map[string]string
}

// NewScraperWorker creates a new instance of ScraperWorker.
// It initializes the worker with necessary dependencies and a default HTTP client/headers.
func NewScraperWorker(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer,
	db *storage.Storage, valkey *redis.Client) (*ScraperWorker, error) {
	baseWorker, err := workers.NewBaseWorker(nc, logger, tracer)
	if err != nil {
		return nil, err
	}

	// Initialize publisher with a descriptive name for clear tracing.
	pub := publishers.NewPublisher(
		fmt.Sprintf("%s-publisher", ScraperWorkerSource),
		baseWorker.JS, baseWorker.Logger, tracer)
	return &ScraperWorker{
		BaseWorker: *baseWorker,
		storage:    db,
		valkey:     valkey,
		publisher:  pub,
		httpCli:    &http.Client{Timeout: 30 * time.Second}, // Default HTTP client with timeout.
		headers: map[string]string{ // Default headers to mimic a real browser.
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
			"Accept-Encoding": "gzip",
			"Accept-Language": "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7",
			"Connection":      "keep-alive",
		},
	}, nil
}

func (w *ScraperWorker) Subject() string {
	return ScraperWorkerSubject
}

func (w *ScraperWorker) StreamName() string {
	return ScraperWorkerStreamName
}

func (w *ScraperWorker) DurableName() string {
	return ScraperWorkerDurableName
}

func (w *ScraperWorker) ConsumerOptions() []nats.SubOpt {
	return []nats.SubOpt{
		nats.DeliverNew(),
		nats.AckExplicit(),
		// Process one message at a time for controlled resource usage.
		nats.MaxAckPending(1),
		nats.ManualAck(),
	}
}

// log is a standardized logging helper that builds upon the BaseWorker.Log method
// to add fields specific to this worker.
func (w ScraperWorker) log(cmd workers.CmdScrapeArticle,
	lvl zerolog.Level, msg string, start time.Time, err error, attrs map[string]any) {
	event := w.BaseWorker.Log(cmd.BaseMessage, lvl, start, attrs)
	event.Err(err).
		Str("url", cmd.URL)
	event.Msg(msg)
}

// Handle processes a single NATS message to scrape an article.
// It orchestrates fetching, parsing, and storing the article,
// and publishes a message upon completion.
func (w *ScraperWorker) Handle(ctx context.Context, msg *nats.Msg) error {
	now := time.Now()
	w.Logger.Info().Msg("ScraperWorker received message")

	// 1. Parse and validate the incoming message.
	var cmd workers.CmdScrapeArticle
	if err := json.Unmarshal(msg.Data, &cmd); err != nil {
		// This is a permanent "poison pill" error. Signal the runner to discard it.
		w.log(cmd, zerolog.ErrorLevel, "malformed message", now, err, map[string]any{
			"message": string(msg.Data),
		})
		return fmt.Errorf("%w: %s", workers.ErrMalformedMessage, err)
	}

	// 2. Fetch Article via HTTP Request.
	var resp *http.Response
	err := func(ctx context.Context) error {
		sCtx, sSpan := w.Tracer.Start(ctx, ScraperWorkerSpanFetch)
		defer sSpan.End()

		select {
		case <-sCtx.Done():
			sSpan.RecordError(sCtx.Err())
			return sCtx.Err()
		default:
			req, err := http.NewRequestWithContext(sCtx, http.MethodGet, cmd.URL, nil)
			if err != nil {
				err = fmt.Errorf("failed to create request: %w", err)
				sSpan.RecordError(err)
				return err
			}

			for k, v := range w.headers {
				req.Header.Set(k, v)
			}

			resp, err = w.httpCli.Do(req)
			if err != nil {
				err = fmt.Errorf("failed to fetch article: %w", err)
				sSpan.RecordError(err)
				return err
			}

			if resp == nil {
				err = fmt.Errorf("http response is nil")
				sSpan.RecordError(err)
				return err
			}
			return nil
		}
	}(ctx)
	if err != nil {
		w.log(cmd, zerolog.ErrorLevel, "failed to fetch article", now, err, nil)
		return err // Propagate the error up to be NAK'd by the runner.
	}

	// 3. Parse the HTTP response body.
	var newsArticle *scrapers.YahooNewsArticle
	err = func(ctx context.Context) error {
		pCtx, pSpan := w.Tracer.Start(ctx, ScraperWorkerSpanParse)
		defer pSpan.End()

		select {
		case <-pCtx.Done():
			pSpan.RecordError(pCtx.Err())
			return pCtx.Err()
		default:
			parseResult := scrapers.ParseYahooNewsResp(resp)
			if parseResult.Error != nil {
				pSpan.RecordError(parseResult.Error)
				return parseResult.Error
			}
			newsArticle = &parseResult.Article
			return nil
		}
	}(ctx)
	if err != nil {
		w.log(cmd, zerolog.ErrorLevel, "failed to parse article response", now, err, nil)
		return fmt.Errorf("failed to parse article response: %w", err)
	}

	// 4. Insert the parsed article into the database.
	var aID int32
	var content string
	cachekey := fmt.Sprintf("%s.article.content", cmd.TaskID.String())
	err = func(ctx context.Context) error {
		iCtx, iSpan := w.Tracer.Start(ctx, ScraperWorkerSpanInsertDB)
		defer iSpan.End()

		// Pre-calculate cumulative lengths of content parts for storage.
		cuts := make([]int32, len(newsArticle.Content))
		cLen := int32(0)
		for i, c := range newsArticle.Content {
			cLen += int32(len(c))
			cuts[i] = cLen
		}
		content = strings.Join(newsArticle.Content, "")

		// Insert into DB. The publisher is passed in to ensure the completion event
		// is sent within the same database transaction for consistency. This guarantees
		// that the NATS message is only published if the article is successfully
		// committed to the database.
		aID, err = w.storage.UserArticles().Insert(iCtx, cmd.TaskID, newsArticle.Title,
			newsArticle.Publisher, content, cuts, newsArticle.Published,
			func(ctx context.Context, tID uuid.UUID, aID int32) error {
				return w.publisher.PublishNATSMessage(ctx, workers.ArticleScraped,
					workers.MsgArticleScraped{
						BaseMessageWithElapsed: workers.BaseMessageWithElapsed{
							BaseMessage: workers.BaseMessage{
								TaskID:   cmd.TaskID,
								EventAt:  now.Unix(),
								Version:  workers.MessageVersion,
								CacheKey: cachekey,
							},
							ElapsedMs: time.Since(now).Milliseconds(),
						},
						ArticleID: aID,
					})
			},
		)
		if err != nil {
			iSpan.RecordError(err)
			return fmt.Errorf("failed to insert article into database: %w", err)
		}
		return nil
	}(ctx)
	if err != nil {
		w.log(cmd, zerolog.ErrorLevel, "failed to insert article into database", now, err, nil)
		return fmt.Errorf("failed to insert article into database: %w", err)
	}

	// 5. Insert the article content into the cache for quick access by the next worker.
	cCtx, cSpan := w.Tracer.Start(ctx, ScraperWorkerSpanInsertCache)
	defer cSpan.End()

	err = w.valkey.Set(cCtx, cachekey, content, time.Hour*3).Err()
	if err != nil {
		cSpan.RecordError(err)
		// A cache failure is not ideal, but the task has succeeded since the data is in the DB.
		// Log a warning but return nil to ACK the message and prevent a pointless retry.
		w.log(cmd, zerolog.WarnLevel,
			"article scraped and inserted successfully, but failed to insert into cache",
			now, err, map[string]any{"article_id": aID})
		return nil
	}

	w.log(cmd, zerolog.InfoLevel,
		"article scraped and inserted successfully",
		now, nil, map[string]any{"article_id": aID})
	return nil
}
