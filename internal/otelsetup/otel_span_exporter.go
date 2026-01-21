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
	"github.com/solarwinds/apm-go/internal/proxy"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

func CreateAndSetupOtelExporter(ctx context.Context) (trace.SpanExporter, error) {
	exporterEndpoint := getAndSetupExporterEndpoint("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	exporterOptions := []otlptracegrpc.Option{}
	gprcDialOptions := []grpc.DialOption{}

	if proxyUrl := config.GetProxy(); proxyUrl != "" {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithEndpoint(proxy.ReplaceSchemeWithPassthrough(exporterEndpoint)))
		gprcDialOptions = append(gprcDialOptions, grpc.WithContextDialer(proxy.NewGRPCProxyDialer(proxy.ProxyOptions{
			Proxy:         proxyUrl,
			ProxyCertPath: config.GetProxyCertPath(),
		})))
	}

	if isExportingToSwo(exporterEndpoint) && !hasAuthorizationHeaderSet() {
		gprcDialOptions = append(gprcDialOptions, grpc.WithPerRPCCredentials(&bearerTokenAuthCred{token: config.GetApiToken()}))
	}

	if len(gprcDialOptions) > 0 {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithDialOption(gprcDialOptions...))
	}

	if os.Getenv("OTEL_EXPORTER_OTLP_COMPRESSION") != "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_COMPRESSION") != "" {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithCompressor("gzip"))
	}

	return otlptracegrpc.New(ctx, exporterOptions...)
}
