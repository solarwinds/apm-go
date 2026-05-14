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

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/otelsetup"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
)

type MetricsPublisher struct {
	metricsRegistry metrics.MetricRegistry
	meterProvider   *metric.MeterProvider
}

func NewMetricsPublisher() *MetricsPublisher {
	return &MetricsPublisher{}
}

func newMeterProvider(ctx context.Context, resource *sdkresource.Resource, runtimeMetrics bool) (*metric.MeterProvider, error) {
	readerOpts := []metric.PeriodicReaderOption{}
	if runtimeMetrics {
		readerOpts = append(readerOpts, metric.WithProducer(runtime.NewProducer()))
	}

	// CreateAndSetupOtelMetricsReader uses autoexport.NewMetricReader with a fallback
	// to CreateAndSetupOtelMetricsExporter when OTEL_METRICS_EXPORTER is unset/empty.
	// Note: readerOpts (including the runtime producer) only apply on the fallback path.
	// When OTEL_METRICS_EXPORTER is set, the user must configure runtime metrics producers
	// via OTEL_METRICS_PRODUCERS or their own SDK setup.
	// runtime.NewProducer produces metrics such as `go.schedule.duration`.
	otelMetricReader, err := otelsetup.CreateAndSetupOtelMetricsReader(ctx, readerOpts...)
	if err != nil {
		return nil, err
	}

	return metric.NewMeterProvider(
		metric.WithReader(otelMetricReader),
		metric.WithResource(resource),
	), nil
}

func (c *MetricsPublisher) ConfigureAndStart(ctx context.Context, o oboe.Oboe, resource *sdkresource.Resource) error {
	runtimeMetricsEnabled := config.GetRuntimeMetrics()
	meterProvider, err := newMeterProvider(ctx, resource, runtimeMetricsEnabled)
	if err != nil {
		return err
	}

	if err = o.RegisterOtelSampleRateMetrics(meterProvider); err != nil {
		return err
	}
	// Register OpenTelemetry contrib runtime metrics
	if runtimeMetricsEnabled {
		if err = runtime.Start(runtime.WithMeterProvider(meterProvider)); err != nil {
			return err
		}
	}
	otel.SetMeterProvider(meterProvider)

	c.metricsRegistry, err = metrics.NewOtelRegistry(meterProvider)
	if err != nil {
		return err
	}
	c.meterProvider = meterProvider

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
	return err
}
