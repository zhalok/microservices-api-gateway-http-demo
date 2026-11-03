package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"ticket-service/internal/httperr"
)

var tracer = otel.Tracer("ticket-service")

// Init configures the global tracer provider. The endpoint comes from
// OTEL_EXPORTER_OTLP_ENDPOINT (e.g. http://jaeger:4318). Returns a shutdown func.
func Init(ctx context.Context) (func(context.Context) error, error) {
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	name := os.Getenv("OTEL_SERVICE_NAME")
	if name == "" {
		name = "ticket-service"
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewSchemaless(semconv.ServiceName(name))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{}) // so trace context crosses service calls
	tracer = tp.Tracer("ticket-service")
	return tp.Shutdown, nil
}

// WithSpan runs fn inside a new span that is a child of the span in ctx.
// Expected 4xx errors are recorded but not marked as span failures.
func WithSpan[T any](ctx context.Context, name string, attrs []attribute.KeyValue, fn func(context.Context) (T, error)) (T, error) {
	ctx, span := tracer.Start(ctx, name, trace.WithAttributes(attrs...))
	defer span.End()
	v, err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		if httperr.Status(err) >= 500 {
			span.SetStatus(codes.Error, err.Error())
		}
	}
	return v, err
}
