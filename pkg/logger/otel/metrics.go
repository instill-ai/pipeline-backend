package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
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

var syncOnce sync.Once
var pipelineSyncTriggerCounter metric.Int64Counter

func SetupSyncTriggerCounter() metric.Int64Counter {
	syncOnce.Do(func() {
		pipelineSyncTriggerCounter, _ = otel.Meter("pipeline.backend").Int64Counter(
			"pipeline.sync.trigger.counter",
			metric.WithUnit("1"),
			metric.WithDescription("user billable action"),
		)
	})

	return pipelineSyncTriggerCounter
}

var asyncOnce sync.Once
var pipelineAsyncTriggerCounter metric.Int64Counter

func SetupAsyncTriggerCounter() metric.Int64Counter {
	asyncOnce.Do(func() {
		pipelineAsyncTriggerCounter, _ = otel.Meter("pipeline.backend").Int64Counter(
			"pipeline.async.trigger.counter",
			metric.WithUnit("1"),
			metric.WithDescription("user billable action"),
		)
	})

	return pipelineAsyncTriggerCounter
}
