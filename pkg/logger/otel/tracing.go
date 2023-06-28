package otel

import (
	"context"
	"fmt"
	"io"

	"github.com/instill-ai/pipeline-backend/config"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func SetupTracing(ctx context.Context, serviceName string) (*trace.TracerProvider, error) {
	var exporter trace.SpanExporter
	var err error
	if config.Config.Log.External {
		exporter, err = otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%s", config.Config.Log.OtelCollector.Host, config.Config.Log.OtelCollector.Port)),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
	} else {
		exporter, err = stdouttrace.New(
			stdouttrace.WithWriter(io.Discard),
		)
		if err != nil {
			return nil, err
		}
	}

	// labels/tags/resources that are common to all traces.
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
	)

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource),
	)

	otel.SetTracerProvider(provider)

	propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
	otel.SetTextMapPropagator(propagator)

	return provider, nil
}
