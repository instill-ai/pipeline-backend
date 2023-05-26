package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
		attribute.String("metrics-common", "common-value"),
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

func SetupTriggerCounterObserver() metric.Int64Counter {
	counter, _ := otel.Meter("pipeline.backend").Int64Counter(
		"vdp.trigger-pipeline.counter_observer",
		metric.WithUnit("1"),
		metric.WithDescription("user billable action"),
	)

	return counter
}

// func SetupTriggerCounterObserver(ctx context.Context) {
// 	meter := global.MeterProvider().Meter("pipeline.backend")
// 	counter, _ := meter.Int64ObservableCounter(
// 		"trigger.pipeline.counter_observer",
// 		metric.WithUnit("1"),
// 		metric.WithDescription("user billable action"),
// 	)

// 	var number int64
// 	if _, err := meter.RegisterCallback(
// 		func(_ context.Context, o metric.Observer) error {
// 			number++
// 			o.ObserveInt64(counter, number)

// 			return nil
// 		},
// 		counter,
// 	); err != nil {
// 		panic(err)
// 	}
// }
