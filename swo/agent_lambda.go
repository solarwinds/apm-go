// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
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

package swo

import (
	"context"
	"os"

	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/otelsetup"
	"github.com/solarwinds/apm-go/internal/processor"
	"github.com/solarwinds/apm-go/internal/propagator"
	"github.com/solarwinds/apm-go/internal/sampler"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Flusher interface {
	Flush(ctx context.Context) error
}

type lambdaFlusher struct {
	Reader *metric.PeriodicReader
}

func (l lambdaFlusher) Flush(ctx context.Context) error {
	return l.Reader.ForceFlush(ctx)
}

var _ Flusher = &lambdaFlusher{}

func StartLambda(lambdaLogStreamName string) (Flusher, error) {
	// By default, the Go OTEL SDK sets this to `https://localhost:4317`, however
	// we do not use https for the local collector in Lambda. We override if not
	// already set.
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		if err := os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"); err != nil {
			log.Warningf("could not override unset OTEL_EXPORTER_OTLP_ENDPOINT %s", err)
		}
	}
	ctx := context.Background()
	o := oboe.NewOboe()
	settingsWatcher := oboe.NewFileBasedWatcher(o)
	// settingsWatcher is started but never stopped in Lambda
	settingsWatcher.Start()
	var err error
	var tpOpts []sdktrace.TracerProviderOption
	var metExp metric.Exporter
	if metExp, err = otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithTemporalitySelector(otelsetup.MetricTemporalitySelector),
	); err != nil {
		return nil, err
	}
	// The reader is flushed manually
	reader := metric.NewPeriodicReader(metExp)
	// The flusher is called after every invocation. We only need to flush
	// metrics here because traces are sent synchronously.
	flusher := &lambdaFlusher{
		Reader: reader,
	}
	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)
	if err = o.RegisterOtelSampleRateMetrics(mp); err != nil {
		return nil, err
	}
	// Register OpenTelemetry contrib runtime metrics
	if err = runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return nil, err
	}
	if exprtr, err := otlptracegrpc.New(ctx); err != nil {
		return nil, err
	} else {
		// Use WithSyncer to flush all spans each invocation
		tpOpts = append(tpOpts, sdktrace.WithSyncer(exprtr))
	}
	registry, err := metrics.NewOtelRegistry(mp)
	if err != nil {
		return nil, err
	}
	proc := processor.NewInboundMetricsSpanProcessor(registry)
	prop := propagation.NewCompositeTextMapPropagator(
		&propagation.TraceContext{},
		&propagation.Baggage{},
		&propagator.SolarwindsPropagator{},
	)
	smplr, err := sampler.NewSampler(o)
	if err != nil {
		return nil, err
	}
	otel.SetTextMapPropagator(prop)
	// Default resource detection plus our required attributes
	var resrc *resource.Resource
	resrc, err = resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("sw.data.module", "apm"),
			attribute.String("sw.apm.version", utils.Version()),
			attribute.String("faas.instance", lambdaLogStreamName),
		),
	)
	if err != nil {
		return nil, err
	}

	tpOpts = append(tpOpts,
		sdktrace.WithResource(resrc),
		sdktrace.WithSampler(smplr),
		sdktrace.WithSpanProcessor(proc),
	)
	otel.SetTracerProvider(sdktrace.NewTracerProvider(tpOpts...))
	return flusher, nil
}
