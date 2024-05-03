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
	"fmt"
	"github.com/pkg/errors"
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/rand"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"time"
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

// a noop reporter
type nullReporter struct{}

func newNullReporter() *nullReporter                      { return &nullReporter{} }
func (r *nullReporter) ReportEvent(Event) error           { return nil }
func (r *nullReporter) ReportStatus(Event) error          { return nil }
func (r *nullReporter) Shutdown(context.Context) error    { return nil }
func (r *nullReporter) ShutdownNow()                      {}
func (r *nullReporter) Closed() bool                      { return true }
func (r *nullReporter) WaitForReady(context.Context) bool { return true }
func (r *nullReporter) SetServiceKey(string) error        { return nil }
func (r *nullReporter) GetServiceName() string            { return "" }

func Start(rsrc *resource.Resource, registry interface{}, o oboe.Oboe) (Reporter, error) {
	log.SetLevelFromStr(config.DebugLevel())
	if reg, ok := registry.(metrics.LegacyRegistry); !ok {
		return nil, fmt.Errorf("metrics registry must implement metrics.LegacyRegistry")
	} else {
		rptr := initReporter(rsrc, reg, o)
		sendInitMessage(rptr, rsrc)
		return rptr, nil
	}
}

func initReporter(r *resource.Resource, registry metrics.LegacyRegistry, o oboe.Oboe) Reporter {
	var rt string
	if !config.GetEnabled() {
		log.Warning("SolarWinds Observability APM agent is disabled.")
		rt = "none"
	} else {
		rt = config.GetReporterType()
	}
	otelServiceName := ""
	if sn, ok := r.Set().Value(semconv.ServiceNameKey); ok {
		otelServiceName = strings.TrimSpace(sn.AsString())
	}
	if rt == "none" {
		return newNullReporter()
	}
	return newGRPCReporter(otelServiceName, registry, o)
}

func CreateInitMessage(tid trace.TraceID, r *resource.Resource) Event {
	evt := NewEventWithRandomOpID(tid, time.Now())
	evt.SetLabel(LabelUnset)
	for _, kv := range r.Attributes() {
		if kv.Key != semconv.ServiceNameKey {
			evt.AddKV(kv)
		}
	}

	evt.AddKVs([]attribute.KeyValue{
		attribute.Bool("__Init", true),
		attribute.String("APM.Version", utils.Version()),
	})
	return evt
}

func sendInitMessage(r Reporter, rsrc *resource.Resource) {
	if r.Closed() {
		log.Info(errors.Wrap(ErrReporterIsClosed, "send init message"))
		return
	}
	tid := trace.TraceID{0}
	rand.Random(tid[:])
	evt := CreateInitMessage(tid, rsrc)
	if err := r.ReportStatus(evt); err != nil {
		log.Error("could not send init message", err)
	}
}
