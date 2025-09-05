// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
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

package reporter

import (
	"context"
	"time"

	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/otelsetup"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
)

type MetricsPublisher struct {
	reporter        *MetricsReporter
	metricsRegistry metrics.MetricRegistry
	meterProvider   *metric.MeterProvider
}

func NewMetricsPublisher(legacyRegistry metrics.LegacyRegistry) *MetricsPublisher {
	return &MetricsPublisher{metricsRegistry: legacyRegistry}
}

func (c *MetricsPublisher) ConfigureAndStart(ctx context.Context, legacyRegistry metrics.LegacyRegistry, conn *grpcConnection, o oboe.Oboe, resource *sdkresource.Resource) error {
	c.reporter = CreatePeriodicMetricsReporter(ctx, conn, legacyRegistry, o)
	c.reporter.Start()

	otelMetricExporter, err := otelsetup.CreateAndSetupOtelMetricsExporter(ctx)
	if err != nil {
		return err
	}

	extra := sdkresource.NewSchemaless(
		attribute.String("publisher.type", "otel"),
	)
	resource, err = sdkresource.Merge(resource, extra)
	if err != nil {
		return err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(otelMetricExporter,
			metric.WithInterval(1*time.Minute))),
		metric.WithResource(resource),
	)
	if err = o.RegisterOtelSampleRateMetrics(meterProvider); err != nil {
		return err
	}
	// err = metrics.RegisterOtelRuntimeMetrics(meterProvider)
	// if err != nil {
	// 	return err
	// }
	otel.SetMeterProvider(meterProvider)

	otelRegistry, err := metrics.NewOtelRegistry(meterProvider)
	if err != nil {
		return err
	}
	c.meterProvider = meterProvider

	c.metricsRegistry = metrics.NewCompositeRegistry(legacyRegistry, otelRegistry)

	return nil
}

func (c *MetricsPublisher) GetMetricsRegistry() metrics.MetricRegistry {
	return c.metricsRegistry
}

func (c *MetricsPublisher) Shutdown() error {
	var err error
	if c.meterProvider != nil {
		if shutdownErr := c.meterProvider.Shutdown(context.Background()); shutdownErr != nil {
			err = shutdownErr
		}
	}
	if c.reporter != nil {
		c.reporter.Shutdown()
	}
	return err
}
