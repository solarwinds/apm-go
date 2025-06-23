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
	"errors"
	"os"

	"io"
	stdlog "log"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/exporter"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/processor"
	"github.com/solarwinds/apm-go/internal/propagator"
	"github.com/solarwinds/apm-go/internal/reporter"
	"github.com/solarwinds/apm-go/internal/sampler"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	errInvalidLogLevel = errors.New("invalid log level")
)

// SetLogLevel changes the logging level of the library
// Valid logging levels: DEBUG, INFO, WARN, ERROR
func SetLogLevel(level string) error {
	l, ok := log.ToLogLevel(level)
	if !ok {
		return errInvalidLogLevel
	}
	log.SetLevel(l)
	return nil
}

// GetLogLevel returns the current logging level of the library
func GetLogLevel() string {
	return log.LevelStr[log.Level()]
}

// SetLogOutput sets the output destination for the internal logger.
func SetLogOutput(w io.Writer) {
	log.SetOutput(w)
}

// Start bootstraps otel requirements and starts the agent. The given `resourceAttrs` are added to the otel
// `resource.Resource` that is supplied to the otel `TracerProvider`
func Start(resourceAttrs ...attribute.KeyValue) (func(), error) {
	ctx := context.Background()
	resrc, err := createResource(resourceAttrs...)
	if err != nil {
		return func() {
			// return a no-op func so that we don't cause a nil-deref for the end-user
		}, err
	}
	isAppoptics := strings.Contains(strings.ToLower(config.GetCollector()), "appoptics.com")
	registry := metrics.NewLegacyRegistry(isAppoptics)
	o := oboe.NewOboe()

	reporter, err := reporter.Start(resrc, registry, o)
	if err != nil {
		return func() {}, err
	}

	exprtr, err := exporter.NewExporter(ctx, reporter)
	if err != nil {
		return func() {}, err
	}

	smplr, err := sampler.NewSampler(o)
	if err != nil {
		return func() {}, err
	}
	config.Load()
	proc := processor.NewInboundMetricsSpanProcessor(registry)
	prop := propagation.NewCompositeTextMapPropagator(
		&propagation.TraceContext{},
		&propagation.Baggage{},
		&propagator.SolarwindsPropagator{},
	)
	otel.SetTextMapPropagator(prop)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exprtr),
		sdktrace.WithResource(resrc),
		sdktrace.WithSampler(smplr),
		sdktrace.WithSpanProcessor(proc),
	)
	otel.SetTracerProvider(tp)
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			stdlog.Fatal(err)
		}
	}, nil

}

// SetTransactionName sets the transaction name of the current entry span. If set multiple times, the last is used.
// Returns nil on success; Error if the provided name is blank, or we are unable to set the transaction name.
func SetTransactionName(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("invalid transaction name")
	}
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return errors.New("could not obtain OpenTelemetry SpanContext from given context")
	}
	return entryspans.SetTransactionName(sc.TraceID(), name)
}

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
		otlpmetricgrpc.WithTemporalitySelector(metrics.TemporalitySelector),
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
