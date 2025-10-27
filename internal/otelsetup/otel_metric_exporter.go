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

package otelsetup

import (
	"context"
	"os"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
)

func CreateAndSetupOtelMetricsExporter(ctx context.Context) (*otlpmetricgrpc.Exporter, error) {
	exporterEndpoint := getAndSetupExporterEndpoint("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
	exporterOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithTemporalitySelector(MetricTemporalitySelector),
		otlpmetricgrpc.WithCompressor("gzip"),
	}

	if isExportingToSwo(exporterEndpoint) && !hasAuthorizationHeaderSet() {
		grpcOptions := []grpc.DialOption{
			grpc.WithPerRPCCredentials(&bearerTokenAuthCred{token: config.GetApiToken()}),
		}
		exporterOptions = append(exporterOptions, otlpmetricgrpc.WithDialOption(grpcOptions...))
	}

	if os.Getenv("OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION") == "" {
		if err := os.Setenv("OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION", "base2_exponential_bucket_histogram"); err != nil {
			log.Warningf("could not override unset OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION %s", err)
		}
	}

	return otlpmetricgrpc.New(
		ctx,
		exporterOptions...,
	)
}

func MetricTemporalitySelector(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}
