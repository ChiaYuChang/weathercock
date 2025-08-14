package global

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitTraceProvider(endpoint string, ctx context.Context) (func(context.Context) error, error) {
	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	Logger.Info().
		Str("endpoint", endpoint).
		Msg("Initializing OpenTelemetry trace provider")

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Logger.Warn().
		Msg("gRPC connection is using insecure credentials (no TLS). Do not expose this endpoint to the public internet.")

	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"https://opentelemetry.io/schemas/1.34.0",
			semconv.ServiceName("weathercock"),
			semconv.DeploymentEnvironment("development"),
		),
	)
	if err != nil {
		return nil, err
	}

	bsp := trace.NewSimpleSpanProcessor(exporter)
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{}))

	return tp.Shutdown, nil
}
