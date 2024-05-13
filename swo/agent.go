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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"io"
	stdlog "log"
	"strings"

	"github.com/pkg/errors"
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

func createResource(resourceAttrs ...attribute.KeyValue) (*resource.Resource, error) {
	return resource.New(context.Background(),
		resource.WithContainer(),
		resource.WithFromEnv(),
		resource.WithOS(),
		resource.WithProcess(),
		// Process runtime description is not recommended[1] for Go and thus is not added by `WithProcess` above.
		// Example value: go version go1.20.4 linux/arm64
		// [1]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/process.md#go-runtimes
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(resourceAttrs...),
	)
}

// Start bootstraps otel requirements and starts the agent. The given `resourceAttrs` are added to the otel
// `resource.Resource` that is supplied to the otel `TracerProvider`
func Start(resourceAttrs ...attribute.KeyValue) (func(), error) {
	resrc, err := createResource(resourceAttrs...)
	if err != nil {
		return func() {
			// return a no-op func so that we don't cause a nil-deref for the end-user
		}, err
	}
	registry := metrics.NewLegacyRegistry()
	o := oboe.NewOboe()
	_reporter, err := reporter.Start(resrc, registry, o)
	if err != nil {
		return func() {}, err
	}

	exprtr := exporter.NewExporter(_reporter)
	smplr, err := sampler.NewSampler(o)
	if err != nil {
		return func() {}, err
	}
	config.Load()
	isAppoptics := strings.Contains(strings.ToLower(config.GetCollector()), "appoptics.com")
	proc := processor.NewInboundMetricsSpanProcessor(registry, isAppoptics)
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
