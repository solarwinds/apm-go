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
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

func CreateAndSetupOtelExporter(ctx context.Context) (trace.SpanExporter, error) {
	exporterEndpoint := getAndSetupExporterEndpoint("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	exporterOptions := []otlptracegrpc.Option{}

	if os.Getenv("OTEL_EXPORTER_OTLP_COMPRESSION") != "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_COMPRESSION") != "" {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithCompressor("gzip"))
	}

	if isExportingToSwo(exporterEndpoint) && !hasAuthorizationHeaderSet() {
		grpcOptions := []grpc.DialOption{
			grpc.WithPerRPCCredentials(&bearerTokenAuthCred{token: config.GetApiToken()}),
		}
		exporterOptions = append(exporterOptions, otlptracegrpc.WithDialOption(grpcOptions...))
	}

	return otlptracegrpc.New(ctx, exporterOptions...)
}
