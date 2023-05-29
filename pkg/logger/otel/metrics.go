package otel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/instill-ai/pipeline-backend/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func SetupMetrics(ctx context.Context, serviceName string) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(fmt.Sprintf("%s:%s", config.Config.Log.OtelCollector.Host, config.Config.Log.OtelCollector.Port)),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
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

var once sync.Once
var pipelineSyncTriggerCounter metric.Int64Counter
func SetupSyncTriggerCounter() metric.Int64Counter {
	once.Do(func() {
		pipelineSyncTriggerCounter, _ = otel.Meter("pipeline.backend").Int64Counter(
			"pipeline.sync.trigger.counter",
			metric.WithUnit("1"),
			metric.WithDescription("user billable action"),
		)
	})

	return pipelineSyncTriggerCounter
}

var pipelineAsyncTriggerCounter metric.Int64Counter
func SetupAsyncTriggerCounter() metric.Int64Counter {
	once.Do(func() {
		pipelineAsyncTriggerCounter, _ = otel.Meter("pipeline.backend").Int64Counter(
			"pipeline.async.trigger.counter",
			metric.WithUnit("1"),
			metric.WithDescription("user billable action"),
		)
	})

	return pipelineAsyncTriggerCounter
}
