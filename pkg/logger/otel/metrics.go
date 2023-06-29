package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/resource"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"github.com/instill-ai/pipeline-backend/config"
)

func SetupMetrics(ctx context.Context, serviceName string) (*sdkmetric.MeterProvider, error) {
	var exporter sdkmetric.Exporter
	var err error
	if config.Config.Log.External {
		exporter, err = otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpoint(fmt.Sprintf("%s:%s", config.Config.Log.OtelCollector.Host, config.Config.Log.OtelCollector.Port)),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
	} else {
		exporter, err = stdoutmetric.New(
			stdoutmetric.WithEncoder(json.NewEncoder(io.Discard)),
			stdoutmetric.WithoutTimestamps(),
		)
		if err != nil {
			return nil, err
		}
	}

	// labels/tags/resources that are common to all metrics.
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(
			// collects and exports metric data every 10 seconds.
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second)),
		),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}
