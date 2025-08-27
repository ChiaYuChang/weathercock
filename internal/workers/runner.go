package workers

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	ec "github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/rs/zerolog"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Runner manages the lifecycle of a worker, handling subscriptions, message fetching,
// health checks, and graceful shutdown.
type Runner struct {
	nc      *nats.Conn
	js      nats.JetStreamContext
	logger  zerolog.Logger
	tracer  trace.Tracer
	worker  Handler
	options Options
}

// NewRunner creates a new Runner instance.
func NewRunner(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer, w Handler, opts ...Option) (*Runner, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	r := &Runner{
		nc:     nc,
		js:     js,
		logger: logger,
		tracer: tracer,
		worker: w,
		options: Options{
			Timeout:         30 * time.Second,
			HealthCheckPort: 8080,
		},
	}

	for _, opt := range opts {
		opt(&r.options)
	}
	return r, nil
}

// Run starts the worker and blocks until the context is canceled.
func (r *Runner) Run(ctx context.Context) error {
	go r.startHealthCheckServer()

	config := r.worker.ConsumerConfig()
	sub, err := r.js.PullSubscribe(
		r.worker.Subject(),
		config.Durable,
		nats.BindStream(config.Name))

	if err != nil {
		e := ec.ErrNATSServerError.Clone().
			WithDetails("failed to create pull subscription").
			Warp(err)
		return e
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	start := time.Now()
	r.logger.Info().
		Str("subject", r.worker.Subject()).
		Str("durable", config.Durable).
		Str("name", config.Name).
		Msg("runner started, waiting for messages...")

	retry := 0
	for {
		select {
		case <-ctx.Done():
			r.logger.Info().
				Dur("uptime", time.Since(start)).
				Msg("runner shutting down gracefully...")
			sub.Unsubscribe()
			return ctx.Err()
		default:
			msgs, err := sub.Fetch(1, nats.MaxWait(5*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				wait := max(1<<retry, 120)
				r.logger.Error().
					Err(err).
					Int("retry", retry).
					Int("wait", wait).
					Msg("failed to fetch messages")
				time.Sleep(time.Duration(wait) * time.Second)
				retry++
				continue
			}
			for _, msg := range msgs {
				r.processMessage(ctx, msg)
			}
		}
	}
}

// processMessage handles the full lifecycle of a single message, including tracing and ack/nak.
func (r *Runner) processMessage(ctx context.Context, msg *nats.Msg) {
	pCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(msg.Header))
	sCtx, sCancel := r.tracer.Start(pCtx, msg.Subject, trace.WithAttributes(
		attribute.String("nats.subject", msg.Subject),
	))
	defer sCancel.End()

	tCtx, tCancel := context.WithTimeout(sCtx, r.options.Timeout)
	defer tCancel()

	if err := r.worker.Handle(tCtx, msg); err != nil {
		sCancel.RecordError(err)
		sCancel.SetAttributes(attribute.Bool("success", false))

		r.logger.Error().Err(err).Msg("worker handler failed, sending NAK")
		if nakErr := msg.NakWithDelay(10 * time.Second); nakErr != nil {
			r.logger.Error().Err(nakErr).Msg("failed to send NAK")
		}
		return
	}

	if ackErr := msg.Ack(); ackErr != nil {
		sCancel.RecordError(ackErr)
		sCancel.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("ack_error", ackErr.Error()))

		r.logger.Error().Err(ackErr).Msg("failed to send ACK")
		return
	}

	sCancel.SetAttributes(attribute.Bool("success", true))
	r.logger.Info().Msg("message processed and ACKed successfully")
}

// startHealthCheckServer starts the HTTP server for health and metric endpoints.
// It intelligently uses custom handlers if the worker provides them, otherwise uses defaults.
func (r *Runner) startHealthCheckServer() {
	mux := http.NewServeMux()

	// Check if the worker implements the optional Healther interface.
	if h, ok := r.worker.(Healther); ok {
		r.logger.Info().Msg("using custom health check handlers provided by worker")
		mux.HandleFunc("/healthz", h.HealthCheck)
		mux.HandleFunc("/readyz", h.Ready)
	} else {
		r.logger.Info().Msg("using default health check handlers")
		mux.HandleFunc("/healthz", r.defaultHealthCheck)
		mux.HandleFunc("/readyz", r.defaultReadyCheck)
	}

	// Check for the optional Metricker interface.
	if m, ok := r.worker.(Metricker); ok {
		r.logger.Info().Msg("using custom metrics handler provided by worker")
		mux.HandleFunc("/metrics", m.Metric)
	} else {
		r.logger.Info().Msg("using default prometheus metrics handler")
		mux.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	}

	addr := fmt.Sprintf("%s:%d", r.options.HealthCheckHost, r.options.HealthCheckPort)
	r.logger.Info().
		Int("health_check_port", r.options.HealthCheckPort).
		Msg("health check server starting")
	if err := http.ListenAndServe(addr, mux); err != nil {
		r.logger.Error().Err(err).Msg("health check server failed")
	}
}

func (r *Runner) defaultHealthCheck(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(ec.Success.HttpStatusCode)
	_ = ec.Success.MarshalAndWriteTo(w)
}

func (r *Runner) defaultReadyCheck(w http.ResponseWriter, req *http.Request) {
	if !r.nc.IsConnected() {
		e := ec.ErrNATSConnectionFailed
		r.logger.Error().Str("remote_addr", req.RemoteAddr).Err(e).Msg("failed to connect to NATS server")
		w.Header().Add("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(e.HttpStatusCode)
		_ = e.MarshalAndWriteTo(w)
		return
	}

	w.WriteHeader(ec.Success.HttpStatusCode)
	_ = ec.Success.MarshalAndWriteTo(w)
}
