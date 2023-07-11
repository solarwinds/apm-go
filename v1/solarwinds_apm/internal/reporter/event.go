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
// Package reporter provides a low-level API for creating and reporting events for
// distributed tracing with SolarWinds Observability.

package reporter

import (
	"errors"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/host"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/bson"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
)

type event struct {
	metadata oboeMetadata
	bbuf     *bson.Buffer
}

// Label is a required event attribute.
type Label string

// Labels used for reporting events for Layer spans.
const (
	LabelEntry = "entry"
	LabelExit  = "exit"
	LabelInfo  = "info"
	LabelError = "error"
	EdgeKey    = "Edge"
)

const (
	eventHeader = "1"
)

// enums used by sampling and tracing settings
type tracingMode int
type settingType int
type settingFlag uint16
type sampleSource int

// tracing modes
const (
	TRACE_DISABLED tracingMode = iota // disable tracing, will neither start nor continue traces
	TRACE_ENABLED                     // perform sampling every inbound request for tracing
	TRACE_UNKNOWN                     // for cache purpose only
)

// setting types
const (
	TYPE_DEFAULT settingType = iota // default setting which serves as a fallback if no other settings found
	TYPE_LAYER                      // layer specific settings
)

// setting flags offset
const (
	FlagInvalidOffset = iota
	FlagOverrideOffset
	FlagSampleStartOffset
	FlagSampleThroughOffset
	FlagSampleThroughAlwaysOffset
	FlagTriggerTraceOffset
)

// setting flags
const (
	FLAG_OK                    settingFlag = 0x0
	FLAG_INVALID               settingFlag = 1 << FlagInvalidOffset
	FLAG_OVERRIDE              settingFlag = 1 << FlagOverrideOffset
	FLAG_SAMPLE_START          settingFlag = 1 << FlagSampleStartOffset
	FLAG_SAMPLE_THROUGH        settingFlag = 1 << FlagSampleThroughOffset
	FLAG_SAMPLE_THROUGH_ALWAYS settingFlag = 1 << FlagSampleThroughAlwaysOffset
	FLAG_TRIGGER_TRACE         settingFlag = 1 << FlagTriggerTraceOffset
)

// source of the sample value
const (
	SAMPLE_SOURCE_UNSET sampleSource = iota - 1
	SAMPLE_SOURCE_NONE
	SAMPLE_SOURCE_FILE
	SAMPLE_SOURCE_DEFAULT
	SAMPLE_SOURCE_LAYER
)

const (
	maxSamplingRate = config.MaxSampleRate
)

// Enabled returns if the trace is enabled or not.
func (f settingFlag) Enabled() bool {
	return f&(FLAG_SAMPLE_START|FLAG_SAMPLE_THROUGH_ALWAYS) != 0
}

// TriggerTraceEnabled returns if the trigger trace is enabled
func (f settingFlag) TriggerTraceEnabled() bool {
	return f&FLAG_TRIGGER_TRACE != 0
}

func (st settingType) toSampleSource() sampleSource {
	var source sampleSource
	switch st {
	case TYPE_DEFAULT:
		source = SAMPLE_SOURCE_DEFAULT
	case TYPE_LAYER:
		source = SAMPLE_SOURCE_LAYER
	default:
		source = SAMPLE_SOURCE_NONE
	}
	return source
}

// newTracingMode creates a tracing mode object from a string
func newTracingMode(mode config.TracingMode) tracingMode {
	switch mode {
	case config.DisabledTracingMode:
		return TRACE_DISABLED
	case config.EnabledTracingMode:
		return TRACE_ENABLED
	default:
	}
	return TRACE_UNKNOWN
}

func (tm tracingMode) isUnknown() bool {
	return tm == TRACE_UNKNOWN
}

func (tm tracingMode) toFlags() settingFlag {
	switch tm {
	case TRACE_ENABLED:
		return FLAG_SAMPLE_START | FLAG_SAMPLE_THROUGH_ALWAYS | FLAG_TRIGGER_TRACE
	case TRACE_DISABLED:
	default:
	}
	return FLAG_OK
}

func (tm tracingMode) ToString() string {
	switch tm {
	case TRACE_ENABLED:
		return string(config.EnabledTracingMode)
	case TRACE_DISABLED:
		return string(config.DisabledTracingMode)
	default:
		return string(config.UnknownTracingMode)
	}
}

type OpIDOption int

const (
	RandomOpID OpIDOption = iota
	UseMDOpID
)

func oboeEventInit(evt *event, md *oboeMetadata, opt OpIDOption) error {
	if evt == nil || md == nil {
		return errors.New("oboeEventInit got nil args")
	}

	// Metadata initialization
	evt.metadata.Init()

	evt.metadata.taskLen = md.taskLen
	evt.metadata.opLen = md.opLen
	switch opt {
	case UseMDOpID:
		copy(evt.metadata.ids.opID, md.ids.opID)
	case RandomOpID:
		if err := evt.metadata.SetRandomOpID(); err != nil {
			return err
		}
	default:
		return errors.New("invalid OpIDOption")
	}
	copy(evt.metadata.ids.taskID, md.ids.taskID)
	evt.metadata.flags = md.flags

	// Buffer initialization
	evt.bbuf = bson.NewBuffer()

	// Copy header to buffer
	evt.bbuf.AppendString("_V", eventHeader)

	// For now the version and flags are always 00 and 01, respectively
	swTraceContext := fmt.Sprintf("00-%x-%x-01", evt.metadata.ids.taskID[:16], evt.metadata.ids.opID[:8])
	evt.bbuf.AppendString("sw.trace_context", swTraceContext)
	// Pack metadata
	mdStr, err := evt.metadata.ToString()
	if err != nil {
		return err
	}
	evt.bbuf.AppendString("X-Trace", mdStr)
	return nil
}

func metaFromSpanContext(ctx trace.SpanContext) *oboeMetadata {
	md := &oboeMetadata{}
	md.Init()
	if ctx.IsSampled() {
		md.flags |= 0x01
	}
	traceID := ctx.TraceID()
	spanID := ctx.SpanID()
	copy(md.ids.taskID, traceID[:])
	copy(md.ids.opID, spanID[:])
	return md
}

func CreateEvent(ctx trace.SpanContext, t time.Time, label Label, opt OpIDOption) (*event, error) {
	md := metaFromSpanContext(ctx)
	evt := &event{}
	if err := oboeEventInit(evt, md, opt); err != nil {
		return nil, err
	}
	evt.AddString("Label", string(label))
	evt.AddInt64("Timestamp_u", t.UnixMicro())
	return evt, nil
}

func CreateEntry(ctx trace.SpanContext, t time.Time, parent trace.SpanContext) (*event, error) {
	evt, err := CreateEvent(ctx, t, LabelEntry, UseMDOpID)
	if err != nil {
		return nil, err
	}
	if parent.IsValid() {
		evt.AddEdgeFromParent(parent)
	}
	return evt, nil
}

func createNonEntryEvent(ctx trace.SpanContext, t time.Time, label Label) (*event, error) {
	evt, err := CreateEvent(ctx, t, label, RandomOpID)
	if err != nil {
		return nil, err
	}
	evt.AddEdgeFromParent(ctx)
	return evt, nil
}

func CreateExit(ctx trace.SpanContext, t time.Time) (*event, error) {
	return createNonEntryEvent(ctx, t, LabelExit)
}

func CreateInfoEvent(ctx trace.SpanContext, t time.Time) (*event, error) {
	return createNonEntryEvent(ctx, t, LabelInfo)
}

func (e *event) AddAttributes(attrs []attribute.KeyValue) {
	for _, kv := range attrs {
		err := e.AddKV(string(kv.Key), kv.Value.AsInterface())
		if err != nil {
			log.Warning("could not add KV", kv, err)
			// Continue so we don't completely abandon the event
		}
	}
}

// Adds string key/value to event. BSON strings are assumed to be Unicode.
func (e *event) AddString(key, value string) { e.bbuf.AppendString(key, value) }

// Adds a binary buffer as a key/value to this event. This uses a binary-safe BSON buffer type.
func (e *event) AddBinary(key string, value []byte) { e.bbuf.AppendBinary(key, value) }

// Adds int key/value to event
func (e *event) AddInt(key string, value int) { e.bbuf.AppendInt(key, value) }

// Adds int64 key/value to event
func (e *event) AddInt64(key string, value int64) { e.bbuf.AppendInt64(key, value) }

// Adds int32 key/value to event
func (e *event) AddInt32(key string, value int32) { e.bbuf.AppendInt32(key, value) }

// Adds float32 key/value to event
func (e *event) AddFloat32(key string, value float32) {
	e.bbuf.AppendFloat64(key, float64(value))
}

// Adds float64 key/value to event
func (e *event) AddFloat64(key string, value float64) { e.bbuf.AppendFloat64(key, value) }

// Adds float key/value to event
func (e *event) AddBool(key string, value bool) { e.bbuf.AppendBool(key, value) }

func (e *event) AddInt64Slice(key string, values []int64) {
	start := e.bbuf.AppendStartArray(key)
	for i, value := range values {
		e.bbuf.AppendInt64(strconv.Itoa(i), value)
	}
	e.bbuf.AppendFinishObject(start)
}

func (e *event) AddStringSlice(key string, values []string) {
	start := e.bbuf.AppendStartArray(key)
	for i, value := range values {
		e.bbuf.AppendString(strconv.Itoa(i), value)
	}
	e.bbuf.AppendFinishObject(start)
}

func (e *event) AddFloat64Slice(key string, values []float64) {
	start := e.bbuf.AppendStartArray(key)
	for i, value := range values {
		e.bbuf.AppendFloat64(strconv.Itoa(i), value)
	}
	e.bbuf.AppendFinishObject(start)
}

func (e *event) AddBoolSlice(key string, values []bool) {
	start := e.bbuf.AppendStartArray(key)
	for i, value := range values {
		e.bbuf.AppendBool(strconv.Itoa(i), value)
	}
	e.bbuf.AppendFinishObject(start)
}

func (e *event) AddEdgeFromParent(parent trace.SpanContext) {
	spanIDHex := parent.SpanID().String()
	e.bbuf.AppendString(EdgeKey, strings.ToUpper(spanIDHex))
	e.bbuf.AppendString("sw.parent_span_id", spanIDHex)
}

// Add any key/value to event. May not add KV if key or value is invalid. Used to facilitate
// reporting variadic args.
func (e *event) AddKV(key, value interface{}) error {
	// load key name
	k, isStr := key.(string)
	if !isStr {
		return fmt.Errorf("key %v (type %T) not a string", k, k)
	}
	// load value and add KV to event
	switch v := value.(type) {
	case string:
		e.AddString(k, v)
	case []byte:
		e.AddBinary(k, v)
	case int:
		e.AddInt(k, v)
	case int64:
		e.AddInt64(k, v)
	case int32:
		e.AddInt32(k, v)
	case uint:
		if v <= math.MaxInt64 {
			e.AddInt64(k, int64(v))
		}
	case uint64:
		if v <= math.MaxInt64 {
			e.AddInt64(k, int64(v))
		}
	case uint32:
		e.AddInt64(k, int64(v))
	case float32:
		e.AddFloat32(k, v)
	case float64:
		e.AddFloat64(k, v)
	case bool:
		e.AddBool(k, v)
	case sampleSource:
		e.AddInt(k, int(v))

	// allow reporting of pointers to basic types as well (for delayed evaluation)
	case *string:
		if v != nil {
			e.AddString(k, *v)
		}
	case *[]byte:
		if v != nil {
			e.AddBinary(k, *v)
		}
	case *int:
		if v != nil {
			e.AddInt(k, *v)
		}
	case *int64:
		if v != nil {
			e.AddInt64(k, *v)
		}
	case *int32:
		if v != nil {
			e.AddInt32(k, *v)
		}
	case *uint:
		if v != nil {
			if *v <= math.MaxInt64 {
				e.AddInt64(k, int64(*v))
			}
		}
	case *uint64:
		if v != nil {
			if *v <= math.MaxInt64 {
				e.AddInt64(k, int64(*v))
			}
		}
	case *uint32:
		if v != nil {
			e.AddInt64(k, int64(*v))
		}
	case *float32:
		if v != nil {
			e.AddFloat32(k, *v)
		}
	case *float64:
		if v != nil {
			e.AddFloat64(k, *v)
		}
	case *bool:
		if v != nil {
			e.AddBool(k, *v)
		}
	case []int64:
		if v != nil {
			e.AddInt64Slice(k, v)
		}
	case []string:
		if v != nil {
			e.AddStringSlice(k, v)
		}
	case []float64:
		if v != nil {
			e.AddFloat64Slice(k, v)
		}
	case []bool:
		if v != nil {
			e.AddBoolSlice(k, v)
		}
	default:
		log.Debugf("Ignoring unrecognized Event key %v val %v valType %T", k, v, v)
	}
	return nil
}

type evType int

const (
	evTypeEvent = iota
	evTypeStatus
)

func report(e *event, typ evType) error {
	if typ != evTypeEvent && typ != evTypeStatus {
		return errors.New("invalid evType")
	}

	e.AddString("Hostname", host.Hostname())
	e.AddInt("PID", host.PID())

	e.bbuf.Finish()
	if typ == evTypeEvent {
		return globalReporter.enqueueEvent(e)
	} else {
		return globalReporter.enqueueStatus(e)
	}
}

func ReportStatus(e *event) error {
	return report(e, evTypeStatus)
}

func ReportEvent(e *event) error {
	return report(e, evTypeEvent)
}
