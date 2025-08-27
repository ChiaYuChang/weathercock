package workers

import (
	"context"
	"net/http"

	"github.com/nats-io/nats.go"
)

// Handler defines the core business logic contract that all workers must implement.
// This is the only mandatory interface for a worker.
type Handler interface {
	// Subject returns the NATS subject the worker will subscribe to.
	Subject() string

	// ConsumerConfig returns the JetStream consumer configuration for the subscription.
	ConsumerConfig() *nats.ConsumerConfig

	// Handle processes a single NATS message. This is where the core business logic resides.
	// It receives a context that includes tracing and timeout information.
	// If an error is returned, the Runner will automatically NAK the message.
	Handle(ctx context.Context, msg *nats.Msg) error
}

// Healther is an optional interface for workers that need custom health check endpoints.
// If a worker implements this interface, the Runner will use its methods instead of the defaults.
type Healther interface {
	HealthCheck(w http.ResponseWriter, r *http.Request)
	Ready(w http.ResponseWriter, r *http.Request)
}

// Metricker is an optional interface for workers that need a custom metrics endpoint.
type Metricker interface {
	Metric(w http.ResponseWriter, r *http.Request)
}
