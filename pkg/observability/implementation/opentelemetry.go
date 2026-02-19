package implementation

import (
	"context"
	"time"

	"github.com/jt828/go-grpc-template/pkg/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type otelTracer struct {
	tracer trace.Tracer
}

type otelSpan struct {
	span trace.Span
}

func (s otelSpan) End()                  { s.span.End() }
func (s otelSpan) RecordError(err error) { s.span.RecordError(err) }

func (t otelTracer) Start(
	ctx context.Context,
	name string,
) (context.Context, observability.Span) {
	ctx, span := t.tracer.Start(ctx, name)
	return ctx, otelSpan{span}
}

func NewOtelTracer(
	ctx context.Context,
	serviceName string,
) (observability.Tracer, func(ctx context.Context) error, error) {
	exp, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("service.version", "0.0.1"),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return otelTracer{tracer: otel.Tracer(serviceName)},
		func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return tp.Shutdown(ctx)
		},
		nil
}
