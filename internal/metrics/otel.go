package metrics

import (
	"context"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type otelRegistry struct {
	meterProvider metric.MeterProvider
}

func (o *otelRegistry) RecordSpan(span sdktrace.ReadOnlySpan, isAppoptics bool) {
	// TODO DRY with legacy registry?
	var attrs = []attribute.KeyValue{
		attribute.Bool("sw.is_error", span.Status().Code == codes.Error),
		attribute.String("sw.transaction", utils.GetTransactionName(span)),
	}
	for _, attr := range span.Attributes() {
		// TODO use semconv?
		if attr.Key == semconv.HTTPMethodKey {
			attrs = append(attrs, attribute.String("http.method", attr.Value.AsString()))
		} else if attr.Key == semconv.HTTPStatusCodeKey {
			attrs = append(attrs, attribute.Int64("http.status_code", attr.Value.AsInt64()))
		} else if attr.Key == semconv.HTTPRouteKey {
			attrs = append(attrs, attribute.String("http.route", attr.Value.AsString()))
		}
	}
	// TODO service.name?
	meter := o.meterProvider.Meter("sw.apm.request.metrics")
	histo, err := meter.Int64Histogram(
		"trace.service.response_time",
		metric.WithExplicitBucketBoundaries(),
		metric.WithUnit("ms"),
	)
	if err != nil {
		log.Error(err)
	} else {
		duration := span.EndTime().Sub(span.StartTime())
		histo.Record(
			context.Background(),
			duration.Milliseconds(),
			metric.WithAttributes(attrs...),
		)
	}
}

var _ MetricRegistry = &otelRegistry{}

func NewOtelRegistry() MetricRegistry {
	return &otelRegistry{}
}

func TemporalitySelector(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}
