// Package workers provides common interfaces, structs, and utilities for building NATS-based workers.
package workers

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// BaseWorker provides a convenient struct with common dependencies that can be
// embedded into concrete worker implementations to reduce boilerplate.
type BaseWorker struct {
	JS     nats.JetStreamContext
	Logger zerolog.Logger
	Tracer trace.Tracer
}

// Log creates a new zerolog.Event with a set of common, standardized fields
// derived from the message's BaseMessage. This promotes consistent, structured logging
// across all workers. It returns the event pointer to allow for method chaining.
func (w BaseWorker) Log(cmd BaseMessage, lvl zerolog.Level, start time.Time, attrs map[string]any) *zerolog.Event {
	event := w.Logger.WithLevel(lvl)
	event.Str("task_id", cmd.TaskID.String()).
		Str("cache_key", cmd.CacheKey).
		Int32("user_id", cmd.UserID).
		Int64("elapsed_ms", time.Since(start).Microseconds())

	for k, v := range attrs {
		switch v := v.(type) {
		case string:
			event.Str(k, v)
		case int:
			event.Int(k, v)
		default:
			event.Interface(k, v)
		}
	}
	return event
}

// NewBaseWorker creates a new instance of BaseWorker.
func NewBaseWorker(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer) (*BaseWorker, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream: %w", err)
	}

	return &BaseWorker{
		JS:     js,
		Logger: logger,
		Tracer: tracer,
	}, nil
}
