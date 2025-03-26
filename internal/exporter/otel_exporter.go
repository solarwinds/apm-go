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

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

func CreateAndSetupOtelExporter(ctx context.Context) (trace.SpanExporter, error) {
	setupOtelExporterEnvironment()

	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithHeaders(map[string]string{
			"Authorization": "Bearer " + config.GetApiToken(),
		}),
	)
}

func setupOtelExporterEnvironment() {
	if os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		otelCollectorAddress := config.GetOtelCollector()
		if err := os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", otelCollectorAddress); err != nil {
			log.Warningf("could not override unset OTEL_EXPORTER_OTLP_TRACES_ENDPOINT %s", err)
		} else {
			log.Infof("Setting Otel exporter to: %s", otelCollectorAddress)
		}
	}
}
