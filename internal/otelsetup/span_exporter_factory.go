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
	"github.com/solarwinds/apm-go/internal/log"
	"go.opentelemetry.io/otel/sdk/trace"
)

func NewSpanExporter(ctx context.Context) (trace.SpanExporter, error) {
	if !config.GetEnabled() {
		log.Warning("SolarWinds Observability exporter is disabled.")
		return &noopExporter{}, nil
	}
	return CreateAndSetupOtelExporter(ctx)
}
