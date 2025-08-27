package publishers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
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
	Conn   *nats.Conn
	Js     nats.JetStreamContext
	Tracer trace.Tracer
}

func (p Publisher) PublishNATSMessage(ctx context.Context, subject string,
	payload any, attrs ...attribute.KeyValue) error {
	attrs = append(attrs, attribute.String("subject", subject))
	ctx, span := p.Tracer.Start(ctx, "Publisher.PublishNATSMessage",
		trace.WithAttributes(attrs...),
	)
	defer span.End()

	headers := nats.Header{}
	otel.GetTextMapPropagator().
		Inject(ctx, propagation.HeaderCarrier(headers))

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Retry publishing the message
	for err, retry := p.Conn.PublishMsg(&nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	}), 0; err != nil; retry++ {
		time.Sleep(MinRetryInterval * 1 << time.Duration(retry))
		if retry >= MaxRetryTimes {
			return err
		}
	}
	return nil
}
