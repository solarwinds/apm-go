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

	"github.com/solarwinds/apm-go/internal/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
)

func CreateAndSetupOtelMetricsExporter(ctx context.Context) (*otlpmetricgrpc.Exporter, error) {
	swApmOtelCollector := config.GetOtelCollector()

	exporterOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithTemporalitySelector(MetricTemporalitySelector),
		otlpmetricgrpc.WithCompressor("gzip"),
		otlpmetricgrpc.WithEndpointURL(swApmOtelCollector),
	}
	grpcOptions := []grpc.DialOption{
		grpc.WithPerRPCCredentials(&bearerTokenAuthCred{token: config.GetApiToken()}),
	}
	exporterOptions = append(exporterOptions, otlpmetricgrpc.WithDialOption(grpcOptions...))

	return otlpmetricgrpc.New(
		ctx,
		exporterOptions...,
	)
}

func MetricTemporalitySelector(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}
