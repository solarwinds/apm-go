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

package exporter

import (
	"context"
	"os"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

func CreateAndSetupOtelExporter(ctx context.Context) (trace.SpanExporter, error) {
	exportingToSwo := setupExporterEndpoint()

	exporterOptions := []otlptracegrpc.Option{}

	if os.Getenv("OTEL_EXPORTER_OTLP_COMPRESSION") != "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_COMPRESSION") != "" {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithCompressor("gzip"))
	}

	if exportingToSwo && !hasAuthorizationHeaderSet() {
		grpcOptions := []grpc.DialOption{
			grpc.WithPerRPCCredentials(&bearerTokenAuthCred{token: config.GetApiToken()}),
		}
		exporterOptions = append(exporterOptions, otlptracegrpc.WithDialOption(grpcOptions...))
	}

	return otlptracegrpc.New(ctx, exporterOptions...)
}

func hasAuthorizationHeaderSet() bool {
	if traceHeaders, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_TRACES_HEADERS "); ok {
		if strings.Contains(strings.ToLower(traceHeaders), "authorization") {
			return true
		}
	} else if otlpHeaders, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_HEADERS"); ok {
		if strings.Contains(strings.ToLower(otlpHeaders), "authorization") {
			return true
		}
	}
	return false
}

func setupExporterEndpoint() (isSwo bool) {
	exporterEndpoint := ""
	ok := false

	if exporterEndpoint, ok = os.LookupEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"); ok {
	} else if exporterEndpoint, ok = os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); ok {
	} else {
		exporterEndpoint = config.GetOtelCollector()
		if err := os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", exporterEndpoint); err != nil {
			log.Warningf("could not override unset OTEL_EXPORTER_OTLP_TRACES_ENDPOINT %s", err)
		} else {
			log.Infof("Setting Otel exporter traces endpoint to: %s", exporterEndpoint)
		}
	}

	if strings.Contains(exporterEndpoint, "solarwinds.com") {
		return true
	}
	return false
}
