package publishers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	MinRetryInterval = 500 * time.Millisecond
	MaxRetryTimes    = 5
)

type Publisher struct {
	Name   string
	js     nats.JetStreamContext
	logger zerolog.Logger
	tracer trace.Tracer
}

func NewPublisher(name string, js nats.JetStreamContext,
	logger zerolog.Logger, tracer trace.Tracer) *Publisher {
	return &Publisher{
		Name:   name,
		js:     js,
		logger: logger,
		tracer: tracer,
	}
}

func (p Publisher) PublishNATSMessage(ctx context.Context, subject string,
	payload any, attrs ...attribute.KeyValue) error {
	attrs = append(attrs, attribute.String("subject", subject))
	sCtx, span := p.tracer.Start(ctx, p.Name, trace.WithAttributes(attrs...))
	defer span.End()

	payload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	headers := nats.Header{}
	otel.GetTextMapPropagator().
		Inject(sCtx, propagation.HeaderCarrier(headers))

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	retry := 0
	_, err = p.js.PublishMsg(&nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	})

	for err != nil && retry < MaxRetryTimes {
		sleep := min(10*time.Second, MinRetryInterval*1<<time.Duration(retry))
		p.logger.Warn().
			Int("retry", retry).
			Str("subject", subject).
			Int("n", len(data)).
			Dur("sleep", sleep).
			Err(err).Msg("falied to publish message")
		time.Sleep(sleep)
		retry++
	}

	if err != nil {
		return errors.ErrNATSMsgPublishFailed.Clone().
			Warp(err).
			WithDetails(
				fmt.Sprintf("retry more than %d times", MaxRetryTimes),
				err.Error(),
			)
	}
	return nil
}
