package subscriber

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func AssignAttrsToOTelSpan(span trace.Span, attrs map[string]any) {
	if attrs == nil {
		return
	}

	for k, attr := range attrs {
		switch val := attr.(type) {
		case int:
			span.SetAttributes(attribute.Int(k, val))
		case int32:
			span.SetAttributes(attribute.Int(k, int(val)))
		case int64:
			span.SetAttributes(attribute.Int64(k, val))
		case float32:
			span.SetAttributes(attribute.Float64(k, float64(val)))
		case float64:
			span.SetAttributes(attribute.Float64(k, val))
		case string:
			span.SetAttributes(attribute.String(k, val))
		case []byte:
			span.SetAttributes(attribute.String(k, string(val)))
		case bool:
			span.SetAttributes(attribute.Bool(k, val))
		}
	}
}
