package api

import (
	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/ChiaYuChang/weathercock/internal/workers/publishers"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

const (
	Version = "v1"
)

// Repo provides methods to interact with the database, cache and nats.
type Repo struct {
	Storage   storage.Storage
	Publisher *publishers.Publisher
	Logger    zerolog.Logger
	Tracer    trace.Tracer
}

// NewRepo creates a new instance of Repo with the provided database and cache clients.
func NewRepo(s storage.Storage, publisher *publishers.Publisher,
	logger zerolog.Logger, tracer trace.Tracer) *Repo {
	return &Repo{
		Storage:   s,
		Publisher: publisher,
		Logger:    logger,
		Tracer:    tracer,
	}
}
