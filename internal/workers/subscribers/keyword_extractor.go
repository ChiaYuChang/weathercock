package subscribers

import (
	"context"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/ChiaYuChang/weathercock/internal/workers"
	"github.com/ChiaYuChang/weathercock/internal/workers/publishers"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

const (
	KeywordExtractorWorkerStreamName  = "TASK"
	KeywordExtractorWorkerDurableName = "keyword-extractor-worker"
	KeywordExtractorWorkerSubject     = workers.TaskExtractKeywords
	KeywordExtractorWorkerSource      = "keyword-extractor-worker"
)

type KeywordExtractorWorker struct {
	workers.BaseWorker
	Storage *storage.Storage
	Valkey  *redis.Client
	LLM     llm.LLM
	Pub     *publishers.Publisher
}

func NewKeywordExtractorWorker(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer,
	store *storage.Storage, valkey *redis.Client, llmClient llm.LLM) (*KeywordExtractorWorker, error) {
	baseWorker, err := workers.NewBaseWorker(nc, logger, tracer)
	if err != nil {
		return nil, err
	}
	return &KeywordExtractorWorker{
		BaseWorker: *baseWorker,
		Storage:    store,
		Valkey:     valkey,
		LLM:        llmClient,
		Pub: &publishers.Publisher{
			Conn:   nc,
			Js:     baseWorker.JetStream,
			Tracer: tracer,
		},
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

func (w *KeywordExtractorWorker) ConsumerOptions() []nats.SubOpt {
	return []nats.SubOpt{
		nats.DeliverNew(),
		nats.AckExplicit(),
		nats.MaxAckPending(1),
		nats.ManualAck(),
	}
}

func (w *KeywordExtractorWorker) Handle(ctx context.Context, msg *nats.Msg) error {
	// TODO: Implement keyword extraction logic here
	w.Logger.Info().Msg("KeywordExtractorWorker received message (not yet implemented)")
	return nil
}
