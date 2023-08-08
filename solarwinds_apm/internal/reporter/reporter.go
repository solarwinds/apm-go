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
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/config"
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/swotel/semconv"
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/w3cfmt"
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

// currently used reporter
var globalReporter Reporter = &nullReporter{}

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

func Start(r *resource.Resource) {
	log.SetLevelFromStr(config.DebugLevel())
	initReporter(r)
	sendInitMessage(r)
}

func initReporter(r *resource.Resource) {
	var rt string
	if config.GetDisabled() {
		log.Warning("SolarWinds Observability APM agent is disabled.")
		rt = "none"
	} else {
		rt = config.GetReporterType()
	}
	otelServiceName := ""
	if sn, ok := r.Set().Value(semconv.ServiceNameKey); ok {
		otelServiceName = strings.TrimSpace(sn.AsString())
	}
	setGlobalReporter(rt, otelServiceName)
}

func setGlobalReporter(reporterType string, otelServiceName string) {
	// Close the previous reporter
	if globalReporter != nil {
		globalReporter.ShutdownNow()
	}

	switch strings.ToLower(reporterType) {
	case "none":
		globalReporter = newNullReporter()
	default:
		globalReporter = newGRPCReporter(otelServiceName)
	}
}

// WaitForReady waits until the reporter becomes ready or the context is canceled.
func WaitForReady(ctx context.Context) bool {
	// globalReporter is not protected by a mutex as currently it's only modified
	// from the init() function.
	return globalReporter.WaitForReady(ctx)
}

// Shutdown flushes the metrics and stops the reporter. It blocked until the reporter
// is shutdown or the context is canceled.
func Shutdown(ctx context.Context) error {
	return globalReporter.Shutdown(ctx)
}

// Closed indicates if the reporter has been shutdown
func Closed() bool {
	return globalReporter.Closed()
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

func SetServiceKey(key string) error {
	return globalReporter.SetServiceKey(key)
}

func ReportStatus(e Event) error {
	return globalReporter.ReportStatus(e)
}

func ReportEvent(e Event) error {
	return globalReporter.ReportEvent(e)
}

func GetServiceName() string {
	return globalReporter.GetServiceName()
}
