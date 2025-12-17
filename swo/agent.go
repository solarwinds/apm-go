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

	stdlog "log"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Start bootstraps otel requirements and starts the agent. The given `resourceAttrs` are added to the otel
// `resource.Resource` that is supplied to the otel `TracerProvider`
func Start(resourceAttrs ...attribute.KeyValue) (func(), error) {
	resrc, err := createResource(resourceAttrs...)
	if err != nil {
		return func() {
			// return a no-op func so that we don't cause a nil-deref for the end-user
		}, err
	}
	isAppoptics := strings.Contains(strings.ToLower(config.GetCollector()), "appoptics.com")
	legacyRegistry := metrics.NewLegacyRegistry(isAppoptics)
	o := oboe.NewOboe()

	settingsUpdater, err := oboe.NewSettingsUpdater(o)
	if err != nil {
		log.Error("Failed to create settings updater, ", err)
		return func() {}, err
	}
	settingsUpdater.Start()

	ctx := context.Background()
	conn, err := reporter.CreateGrpcConnection()
	if err != nil {
		log.Error("Failed to create gRPC connection to SWO APM", err)
		return func() {}, err
	}
	backgroundReporter, err := reporter.CreateAndStartBackgroundReporter(conn, resrc, legacyRegistry)
	if err != nil {
		log.Error("Failed to configure and start background reporter", err)
		return func() {}, err
	}

	if conn != nil {
		reporter.CreateAndSendOneTimeInitMessage(backgroundReporter, resrc)
	}

	exprtr, err := exporter.NewSpanExporter(ctx, backgroundReporter)
	if err != nil {
		log.Error("Failed to configure span exporter", err)
		return func() {}, err
	}

	metricsPublisher := reporter.NewMetricsPublisher(legacyRegistry)
	err = metricsPublisher.ConfigureAndStart(ctx, legacyRegistry, conn, o, resrc)
	if err != nil {
		log.Error("Failed to configure and start metrics publisher", err)
		return func() {}, err
	}

	smplr, err := sampler.NewSampler(o)
	if err != nil {
		return func() {}, err
	}
	config.Load()
	proc := processor.NewInboundMetricsSpanProcessor(metricsPublisher.GetMetricsRegistry())
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
		err := metricsPublisher.Shutdown()
		if err != nil {
			log.Error("Failed to Shutdown metrics publisher, ", err)
		}
		err = backgroundReporter.Shutdown(ctx)
		if err != nil {
			log.Error("Failed to Shutdown background reporter, ", err)
		}
		settingsUpdater.Shutdown()
		if err = tp.Shutdown(context.Background()); err != nil {
			stdlog.Fatal(err)
		}
	}, nil
}
