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
	"encoding/binary"
	"fmt"
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/w3cfmt"
	"go.opentelemetry.io/otel/sdk/resource"
	"math"
	"strings"
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

// KVs from getSettingsResult arguments
const (
	kvSignatureKey                      = "SignatureKey"
	kvBucketCapacity                    = "BucketCapacity"
	kvBucketRate                        = "BucketRate"
	kvTriggerTraceRelaxedBucketCapacity = "TriggerRelaxedBucketCapacity"
	kvTriggerTraceRelaxedBucketRate     = "TriggerRelaxedBucketRate"
	kvTriggerTraceStrictBucketCapacity  = "TriggerStrictBucketCapacity"
	kvTriggerTraceStrictBucketRate      = "TriggerStrictBucketRate"
	kvMetricsFlushInterval              = "MetricsFlushInterval"
	kvEventsFlushInterval               = "EventsFlushInterval"
	kvMaxTransactions                   = "MaxTransactions"
	kvMaxCustomMetrics                  = "MaxCustomMetrics"
)

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

func Start(rsrc *resource.Resource, registry interface{}) (Reporter, error) {
	log.SetLevelFromStr(config.DebugLevel())
	if reg, ok := registry.(metrics.LegacyRegistry); !ok {
		return nil, fmt.Errorf("metrics registry must implement metrics.LegacyRegistry")
	} else {
		rptr := initReporter(rsrc, reg)
		sendInitMessage(rptr, rsrc)
		return rptr, nil
	}
}

func initReporter(r *resource.Resource, registry metrics.LegacyRegistry) Reporter {
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
	return newGRPCReporter(otelServiceName, registry)
}

func ShouldTraceRequestWithURL(traced bool, url string, ttMode TriggerTraceMode, swState w3cfmt.SwTraceState) SampleDecision {
	return shouldTraceRequestWithURL(traced, url, ttMode, swState)
}

func shouldTraceRequestWithURL(traced bool, url string, triggerTrace TriggerTraceMode, swState w3cfmt.SwTraceState) SampleDecision {
	return oboeSampleRequest(traced, url, triggerTrace, swState)
}

func argsToMap(capacity, ratePerSec, tRCap, tRRate, tSCap, tSRate float64,
	metricsFlushInterval, maxTransactions int, token []byte) map[string][]byte {
	args := make(map[string][]byte)

	if capacity > -1 {
		bits := math.Float64bits(capacity)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[kvBucketCapacity] = bytes
	}
	if ratePerSec > -1 {
		bits := math.Float64bits(ratePerSec)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[kvBucketRate] = bytes
	}
	if tRCap > -1 {
		bits := math.Float64bits(tRCap)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[kvTriggerTraceRelaxedBucketCapacity] = bytes
	}
	if tRRate > -1 {
		bits := math.Float64bits(tRRate)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[kvTriggerTraceRelaxedBucketRate] = bytes
	}
	if tSCap > -1 {
		bits := math.Float64bits(tSCap)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[kvTriggerTraceStrictBucketCapacity] = bytes
	}
	if tSRate > -1 {
		bits := math.Float64bits(tSRate)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[kvTriggerTraceStrictBucketRate] = bytes
	}
	if metricsFlushInterval > -1 {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(metricsFlushInterval))
		args[kvMetricsFlushInterval] = bytes
	}
	if maxTransactions > -1 {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(maxTransactions))
		args[kvMaxTransactions] = bytes
	}

	args[kvSignatureKey] = token

	return args
}
