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
	"os"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
)

func isExportingToSwo(exporterEndpoint string) bool {
	return strings.Contains(exporterEndpoint, "solarwinds.com")
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

func getAndSetupExporterEndpoint(specificExporterEnvVariable string) string {
	exporterEndpoint := ""
	ok := false
	if exporterEndpoint, ok = os.LookupEnv(specificExporterEnvVariable); ok {
	} else if exporterEndpoint, ok = os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); ok {
	} else {
		swApmOtelCollector := config.GetOtelCollector()
		if err := os.Setenv(specificExporterEnvVariable, swApmOtelCollector); err != nil {
			log.Warningf("could not override unset %s %s", specificExporterEnvVariable, err)
		} else {
			exporterEndpoint = swApmOtelCollector
		}
	}
	log.Infof("Otel span exporter endpoint: %s", exporterEndpoint)

	return exporterEndpoint
}
