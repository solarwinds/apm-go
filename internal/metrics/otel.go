// © 2024 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"github.com/solarwinds/apm-go/internal/txn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type otelRegistry struct {
	histo metric.Int64Histogram
}

var searchSet = map[attribute.Key]bool{
	semconv.HTTPMethodKey:     true,
	semconv.HTTPStatusCodeKey: true,
	semconv.HTTPRouteKey:      true,
}

func (o *otelRegistry) RecordSpan(span sdktrace.ReadOnlySpan) {
	var attrs = []attribute.KeyValue{
		attribute.Bool("sw.is_error", span.Status().Code == codes.Error),
		attribute.String("sw.transaction", txn.GetTransactionName(span)),
	}
	if span.SpanKind() == trace.SpanKindServer {
		for _, attr := range span.Attributes() {
			if searchSet[attr.Key] {
				attrs = append(attrs, attr)
			}
		}
	}
	duration := span.EndTime().Sub(span.StartTime())
	o.histo.Record(
		context.Background(),
		duration.Milliseconds(),
		metric.WithAttributes(attrs...),
	)
}

var _ MetricRegistry = &otelRegistry{}

func NewOtelRegistry(p metric.MeterProvider) (MetricRegistry, error) {
	meter := p.Meter("sw.apm.request.metrics")
	if histo, err := meter.Int64Histogram(
		"trace.service.response_time",
		metric.WithExplicitBucketBoundaries(),
		metric.WithUnit("ms"),
	); err != nil {
		return nil, err
	} else {
		return &otelRegistry{histo: histo}, nil
	}
}

func TemporalitySelector(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}
