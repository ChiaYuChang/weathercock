package workers

import (
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// BaseWorker provides a convenient struct with common dependencies that can be
// embedded into concrete worker implementations to reduce boilerplate.
type BaseWorker struct {
	NatsConn  *nats.Conn
	JetStream nats.JetStreamContext
	Logger    zerolog.Logger
	Tracer    trace.Tracer
}

// NewBaseWorker creates a new instance of BaseWorker.
func NewBaseWorker(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer) (*BaseWorker, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &BaseWorker{
		NatsConn:  nc,
		JetStream: js,
		Logger:    logger,
		Tracer:    tracer,
	}, nil
}
