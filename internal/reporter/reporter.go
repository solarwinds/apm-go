// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
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
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/state"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"go.opentelemetry.io/otel/sdk/resource"
)

// defines what methods a Reporter should offer (internal to Reporter package)
type Reporter interface {
	ReportEvent(e Event) error
	ReportStatus(e Event) error
	// Shutdown closes the Reporter.
	Shutdown(ctx context.Context) error
	// ShutdownNow closes the Reporter immediately, logs on error
	ShutdownNow()
	// Closed returns if the Reporter is already closed.
	Closed() bool
	// WaitForReady waits until the Reporter becomes ready or the context is canceled.
	WaitForReady(context.Context) bool
	// SetServiceKey attaches a service key to the Reporter
	// Returns error if service key is invalid
	SetServiceKey(key string) error
	// GetServiceName retrieves the current service name, preferring an otel `service.name` from resource attributes,
	// falling back to the service name in the service key
	GetServiceName() string
}

var (
	periodicTasksDisabled = false // disable periodic tasks, for testing
)

func CreateAndStartBackgroundReporter(conn *grpcConnection, rsrc *resource.Resource, reg metrics.LegacyRegistry, o oboe.Oboe) (Reporter, error) {
	conn.AddClient()
	log.SetLevelFromStr(config.DebugLevel())
	rptr := initReporter(conn, rsrc, reg, o)
	return rptr, nil
}

func initReporter(conn *grpcConnection, r *resource.Resource, registry metrics.LegacyRegistry, o oboe.Oboe) Reporter {
	otelServiceName := ""
	if sn, ok := r.Set().Value(semconv.ServiceNameKey); ok {
		otelServiceName = strings.TrimSpace(sn.AsString())
		state.SetServiceName(otelServiceName)
	}
	if !config.GetEnabled() {
		log.Warning("SolarWinds Observability APM agent is disabled.")
		return newNullReporter()
	}

	if conn == nil {
		return newNullReporter()
	}

	return newGRPCReporter(conn, otelServiceName, registry, o)
}
