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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/constants"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/host"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	mbson "gopkg.in/mgo.v2/bson"
	"testing"
	"time"
)

var validTraceID = trace.TraceID{0x01}
var validSpanID = trace.SpanID{0x02}

var validSpanContext = trace.NewSpanContext(trace.SpanContextConfig{
	TraceID: validTraceID,
	SpanID:  validSpanID,
})

var invalidSpanID = trace.SpanID{}
var invalidSpanContext = trace.NewSpanContext(trace.SpanContextConfig{
	SpanID: invalidSpanID,
})

func TestEntryEventNoParent(t *testing.T) {
	now := time.Now()
	e := CreateEntryEvent(validSpanContext, now, invalidSpanContext)
	require.NotNil(t, e)

	evt, ok := e.(*event)
	require.True(t, ok)
	require.Equal(t, now, evt.t)
	require.Equal(t, validTraceID, evt.taskID)
	require.Equal(t, validSpanID[:], evt.opID[:])
	require.False(t, evt.parent.IsValid())
	require.Equal(t, invalidSpanID, evt.parent)
	require.Equal(t, LabelEntry, evt.label)
	require.Empty(t, evt.layer)
	require.Nil(t, evt.kvs)
	evt.AddKV(attribute.String("foo", "bar"))

	b := e.ToBson()
	m := make(map[string]interface{})
	require.Nil(t, mbson.Unmarshal(b, m))
	require.Len(t, m, 7)
	require.Equal(t, host.Hostname(), m["Hostname"])
	require.Equal(t, constants.EntryLabel, m["Label"])
	require.Equal(t, host.PID(), m["PID"])
	require.Equal(t, now.UnixMicro(), m["Timestamp_u"])
	require.Equal(t, evt.GetXTrace(), m["X-Trace"])
	require.Equal(t, evt.GetSwTraceContext(), m["sw.trace_context"])
	require.Equal(t, "bar", m["foo"])
}

func TestEntryEventWithParent(t *testing.T) {
	now := time.Now()
	parentSpanID := trace.SpanID{0x33}
	parent := trace.NewSpanContext(trace.SpanContextConfig{TraceID: validTraceID, SpanID: parentSpanID})
	e := CreateEntryEvent(validSpanContext, now, parent)
	require.NotNil(t, e)

	evt, ok := e.(*event)
	require.True(t, ok)
	require.Equal(t, now, evt.t)
	require.Equal(t, validTraceID, evt.taskID)
	require.Equal(t, validSpanID[:], evt.opID[:])
	require.True(t, evt.parent.IsValid())
	require.Equal(t, parentSpanID, evt.parent)
	require.Equal(t, LabelEntry, evt.label)
	require.Empty(t, evt.layer)
	require.Nil(t, evt.kvs)
	evt.AddKV(attribute.String("foo", "bar"))

	b := e.ToBson()
	m := make(map[string]interface{})
	require.Nil(t, mbson.Unmarshal(b, m))
	require.Len(t, m, 9)
	require.Equal(t, "3300000000000000", m["Edge"])
	require.Equal(t, "3300000000000000", m["sw.parent_span_id"])
	require.Equal(t, host.Hostname(), m["Hostname"])
	require.Equal(t, constants.EntryLabel, m["Label"])
	require.Equal(t, host.PID(), m["PID"])
	require.Equal(t, now.UnixMicro(), m["Timestamp_u"])
	require.Equal(t, evt.GetXTrace(), m["X-Trace"])
	require.Equal(t, evt.GetSwTraceContext(), m["sw.trace_context"])
	require.Equal(t, "bar", m["foo"])
}

func TestExitEvent(t *testing.T) {
	now := time.Now()
	e := CreateExitEvent(validSpanContext, now)
	require.NotNil(t, e)

	evt, ok := e.(*event)
	require.True(t, ok)
	require.Equal(t, now, evt.t)
	require.Equal(t, validTraceID, evt.taskID)
	require.NotEqual(t, validSpanID[:], evt.opID[:])
	require.NotEqual(t, invalidSpanID[:], evt.opID[:])
	require.True(t, evt.parent.IsValid())
	require.Equal(t, validSpanID, evt.parent)
	require.Equal(t, LabelExit, evt.label)
	require.Empty(t, evt.layer)
	require.Nil(t, evt.kvs)
	evt.AddKV(attribute.String("foo", "bar"))

	b := e.ToBson()
	m := make(map[string]interface{})
	require.Nil(t, mbson.Unmarshal(b, m))
	require.Len(t, m, 9)
	require.Equal(t, "0200000000000000", m["Edge"])
	require.Equal(t, "0200000000000000", m["sw.parent_span_id"])
	require.Equal(t, host.Hostname(), m["Hostname"])
	require.Equal(t, constants.ExitLabel, m["Label"])
	require.Equal(t, host.PID(), m["PID"])
	require.Equal(t, now.UnixMicro(), m["Timestamp_u"])
	require.Equal(t, evt.GetXTrace(), m["X-Trace"])
	require.Equal(t, evt.GetSwTraceContext(), m["sw.trace_context"])
	require.Equal(t, "bar", m["foo"])
}

func TestInfoEvent(t *testing.T) {
	now := time.Now()
	e := CreateInfoEvent(validSpanContext, now)
	require.NotNil(t, e)

	evt, ok := e.(*event)
	require.True(t, ok)
	require.Equal(t, now, evt.t)
	require.Equal(t, validTraceID, evt.taskID)
	require.NotEqual(t, validSpanID[:], evt.opID[:])
	require.NotEqual(t, invalidSpanID[:], evt.opID[:])
	require.True(t, evt.parent.IsValid())
	require.Equal(t, validSpanID, evt.parent)
	require.Equal(t, LabelInfo, evt.label)
	require.Empty(t, evt.layer)
	require.Nil(t, evt.kvs)
	evt.AddKV(attribute.String("foo", "bar"))

	b := e.ToBson()
	m := make(map[string]interface{})
	require.Nil(t, mbson.Unmarshal(b, m))
	require.Len(t, m, 9)
	require.Equal(t, "0200000000000000", m["Edge"])
	require.Equal(t, "0200000000000000", m["sw.parent_span_id"])
	require.Equal(t, host.Hostname(), m["Hostname"])
	require.Equal(t, constants.InfoLabel, m["Label"])
	require.Equal(t, host.PID(), m["PID"])
	require.Equal(t, now.UnixMicro(), m["Timestamp_u"])
	require.Equal(t, evt.GetXTrace(), m["X-Trace"])
	require.Equal(t, evt.GetSwTraceContext(), m["sw.trace_context"])
	require.Equal(t, "bar", m["foo"])
}

func TestAddKV(t *testing.T) {
	now := time.Now()
	e := CreateExitEvent(validSpanContext, now)
	require.NotNil(t, e)

	evt, ok := e.(*event)
	require.True(t, ok)
	e.AddKV(attribute.String("foo", "bar"))
	e.AddKVs([]attribute.KeyValue{
		attribute.Bool("is_false", false),
		attribute.Int64("numeric", 321),
	})

	require.Equal(t, []attribute.KeyValue{
		attribute.String("foo", "bar"),
		attribute.Bool("is_false", false),
		attribute.Int64("numeric", 321),
	}, evt.kvs)
}

func TestEventXTraceAndSwTraceCtx(t *testing.T) {
	traceID := trace.TraceID{0xAA, 0xBB, 0xCC, 0xDD}
	spanID := trace.SpanID{0xFF, 0xEE, 0xDD, 0xCC}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	e := CreateEntryEvent(sc, time.Now(), invalidSpanContext)
	require.NotNil(t, e)
	evt, ok := e.(*event)
	require.True(t, ok)
	x := evt.GetXTrace()
	require.Len(t, x, 60)
	require.Equal(t, "2BAABBCCDD00000000000000000000000000000000FFEEDDCC0000000001", x)

	s := evt.GetSwTraceContext()
	require.Len(t, s, 55)
	require.Equal(t, "00-aabbccdd000000000000000000000000-ffeeddcc00000000-01", s)
}

func TestLabelAsString(t *testing.T) {
	require.Equal(t, constants.EntryLabel, LabelEntry.AsString())
	require.Equal(t, constants.ErrorLabel, LabelError.AsString())
	require.Equal(t, constants.ExitLabel, LabelExit.AsString())
	require.Equal(t, constants.InfoLabel, LabelInfo.AsString())
	require.Equal(t, constants.UnknownLabel, LabelUnset.AsString())
}
