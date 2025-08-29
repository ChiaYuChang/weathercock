package subscribers

import (
	"context"

	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/ChiaYuChang/weathercock/internal/workers"
	"github.com/ChiaYuChang/weathercock/internal/workers/publishers"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"

	"github.com/nats-io/nats.go"
)

const (
	ScraperWorkerStreamName  = "TASK"
	ScraperWorkerDurableName = "scraper-worker"
	ScraperWorkerSubject     = workers.TaskScrape
	ScraperWorkerSource      = "scraper-worker"
)

// ScraperWorker implements the Handler interface for scraping articles.
type ScraperWorker struct {
	workers.BaseWorker
	Storage *storage.Storage
	Valkey  *redis.Client
	Pub     *publishers.Publisher
}

// NewScraperWorker creates a new instance of ScraperWorker.
func NewScraperWorker(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer,
	db *storage.Storage, valkey *redis.Client) (*ScraperWorker, error) {
	baseWorker, err := workers.NewBaseWorker(nc, logger, tracer)
	if err != nil {
		return nil, err
	}
	return &ScraperWorker{
		BaseWorker: *baseWorker,
		Storage:    db,
		Valkey:     valkey,
		Pub: &publishers.Publisher{
			Conn:   nc,
			Js:     baseWorker.JetStream,
			Tracer: tracer,
		},
	}, nil
}

// Subject returns the NATS subject the worker will subscribe to.
func (w *ScraperWorker) Subject() string {
	return ScraperWorkerSubject
}

// StreamName returns the name of the JetStream stream to bind to.
func (w *ScraperWorker) StreamName() string {
	return ScraperWorkerStreamName
}

// DurableName returns the durable name for the consumer.
func (w *ScraperWorker) DurableName() string {
	return ScraperWorkerDurableName
}

// ConsumerOptions allows for advanced configuration of the consumer.
func (w *ScraperWorker) ConsumerOptions() []nats.SubOpt {
	return []nats.SubOpt{
		nats.DeliverNew(),
		nats.AckExplicit(),
		nats.MaxAckPending(1),
		nats.ManualAck(),
	}
}

// Handle processes a single NATS message to scrape an article.
func (w *ScraperWorker) Handle(ctx context.Context, msg *nats.Msg) error {
	// TODO: Implement scraping logic here
	w.Logger.Info().Msg("ScraperWorker received message (not yet implemented)")
	return nil
}
