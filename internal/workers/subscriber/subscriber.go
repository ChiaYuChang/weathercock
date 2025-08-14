package subscriber

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Subscriber struct {
	Conn   *nats.Conn
	Tracer trace.Tracer
}

type MessageHandler func(ctx context.Context, msg *nats.Msg) (attrs map[string]any, err error)

func (s Subscriber) Subscribe(subject string, handler MessageHandler) (*Worker, error) {
	ch := make(chan *nats.Msg)
	sub, err := s.Conn.ChanSubscribe(subject, ch)
	if err != nil {
		return nil, err
	}

	return &Worker{
		Subscription: sub,
		Chan:         ch,
		Tracer:       s.Tracer,
		Handler:      handler,
		Timeout:      10 * time.Second,
	}, nil
}

type Worker struct {
	Subscription *nats.Subscription
	Chan         chan *nats.Msg
	Tracer       trace.Tracer
	Handler      MessageHandler
	Timeout      time.Duration
}

func (w *Worker) SetTimeout(timeout time.Duration) *Worker {
	w.Timeout = timeout
	return w
}

func (w *Worker) Do() <-chan error {
	ch := make(chan error)
	go func() {
		for msg := range w.Chan {
			func() {
				ctx := otel.GetTextMapPropagator().
					Extract(context.Background(),
						propagation.HeaderCarrier(msg.Header),
					)

				ctx, span := w.Tracer.Start(ctx, w.Subscription.Subject)
				defer span.End()

				start := time.Now()
				ctx, cancel := context.WithTimeout(ctx, w.Timeout)
				defer cancel()

				attrs, err := w.Handler(ctx, msg)
				AssignAttrsToOTelSpan(span, attrs)
				span.SetAttributes(
					attribute.Int64("elapsed_ms", time.Since(start).Milliseconds()),
				)

				if err != nil {
					span.SetAttributes(
						attribute.String("error", err.Error()),
						attribute.Bool("success", false),
					)
					ch <- err
					return
				}

				if err := msg.Ack(); err != nil {
					span.SetAttributes(
						attribute.String("ack_error", err.Error()),
						attribute.Bool("success", false),
					)
					ch <- err
					return
				}

				span.SetAttributes(
					attribute.Bool("success", true),
				)
			}()
		}
	}()
	return ch
}

func (w Worker) Close() error {
	if w.Subscription != nil {
		if err := w.Subscription.Unsubscribe(); err != nil {
			return err
		}
		close(w.Chan)
	}
	return nil
}
