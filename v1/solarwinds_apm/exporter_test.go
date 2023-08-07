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

package solarwinds_apm

import (
	"context"
	"errors"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/entryspans"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/host"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/testutils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	mbson "gopkg.in/mgo.v2/bson"
	"strings"
	"testing"
	"time"
)

func TestExportSpan(t *testing.T) {
	r := &capturingReporter{}
	defer reporter.SetGlobalReporter(r)()
	tr, cb := testutils.TracerWithExporter(NewExporter())
	defer cb()

	ctx := context.Background()
	name := "foo"
	var span trace.Span
	start := time.Now()
	end := start.Add(time.Second)
	infoT := start.Add(time.Millisecond)
	errorT := start.Add(10 * time.Millisecond)
	_, span = tr.Start(ctx, name, trace.WithTimestamp(start))
	span.AddEvent("info event",
		trace.WithAttributes(attribute.String("foo", "bar")),
		trace.WithTimestamp(infoT),
	)
	err := errors.New("this is an error")
	span.RecordError(
		err,
		trace.WithTimestamp(errorT),
	)

	// Set the entryspan as if it were set in the SpanProcessor, and assert
	// it's removed at the end
	require.NoError(t, entryspans.Push(span.(sdktrace.ReadOnlySpan)))
	sid, ok := entryspans.Current(span.SpanContext().TraceID())
	require.True(t, ok)
	require.Equal(t, span.SpanContext().SpanID(), sid)

	span.End(trace.WithTimestamp(end))
	require.Len(t, r.events, 4)

	// Successfully removed in the Exporter
	_, ok = entryspans.Current(span.SpanContext().TraceID())
	require.False(t, ok)

	{ // Entry event
		entry := r.events[0]
		result := make(map[string]interface{})
		require.NoError(t, mbson.Unmarshal(entry.ToBson(), result))

		require.Equal(t, map[string]interface{}{
			"Hostname":           host.Hostname(),
			"Label":              "entry",
			"Language":           "Go",
			"Layer":              "internal:foo",
			"PID":                host.PID(),
			"Timestamp_u":        start.UnixMicro(),
			"TransactionName":    "foo",
			"X-Trace":            entry.GetXTrace(),
			"otel.scope.name":    "foo123",
			"otel.scope.version": "123",
			"sw.span_kind":       "internal",
			"sw.span_name":       "foo",
			"sw.trace_context":   entry.GetSwTraceContext(),
		}, result)
	}
	{ // Info event
		info := r.events[1]
		result := make(map[string]interface{})
		require.NoError(t, mbson.Unmarshal(info.ToBson(), result))

		spanId := span.SpanContext().SpanID().String()
		require.Equal(t, map[string]interface{}{
			"Edge":              strings.ToUpper(spanId),
			"sw.parent_span_id": spanId,
			"Hostname":          host.Hostname(),
			"Label":             "info",
			"PID":               host.PID(),
			"Timestamp_u":       infoT.UnixMicro(),
			"X-Trace":           info.GetXTrace(),
			"foo":               "bar",
			"sw.trace_context":  info.GetSwTraceContext(),
		}, result)
	}

	{ // Error event
		errEvt := r.events[2]
		result := make(map[string]interface{})
		require.NoError(t, mbson.Unmarshal(errEvt.ToBson(), result))

		spanId := span.SpanContext().SpanID().String()
		require.Equal(t, map[string]interface{}{
			"Edge":              strings.ToUpper(spanId),
			"ErrorClass":        "*errors.errorString",
			"exception.type":    "*errors.errorString",
			"ErrorMsg":          "this is an error",
			"exception.message": "this is an error",
			"sw.parent_span_id": spanId,
			"Hostname":          host.Hostname(),
			"Label":             "error",
			"PID":               host.PID(),
			"Timestamp_u":       errorT.UnixMicro(),
			"X-Trace":           errEvt.GetXTrace(),
			"sw.trace_context":  errEvt.GetSwTraceContext(),
		}, result)
	}

	{ // Exit event
		exit := r.events[3]
		result := make(map[string]interface{})
		require.NoError(t, mbson.Unmarshal(exit.ToBson(), result))

		spanId := span.SpanContext().SpanID().String()
		require.Equal(t, map[string]interface{}{
			"Edge":              strings.ToUpper(spanId),
			"sw.parent_span_id": spanId,
			"Hostname":          host.Hostname(),
			"Label":             "exit",
			"Layer":             "internal:foo",
			"PID":               host.PID(),
			"Timestamp_u":       end.UnixMicro(),
			"X-Trace":           exit.GetXTrace(),
			"sw.trace_context":  exit.GetSwTraceContext(),
		}, result)
	}
}

func TestExportSpanBacktrace(t *testing.T) {
	r := &capturingReporter{}
	defer reporter.SetGlobalReporter(r)()
	tr, cb := testutils.TracerWithExporter(NewExporter())
	defer cb()

	ctx := context.Background()
	name := "foo"
	var span trace.Span
	start := time.Now()
	end := start.Add(time.Second)
	errorT := start.Add(10 * time.Millisecond)
	_, span = tr.Start(ctx, name, trace.WithTimestamp(start))
	span.RecordError(
		errors.New("this is an error"),
		trace.WithTimestamp(errorT),
		trace.WithStackTrace(true),
	)
	span.End(trace.WithTimestamp(end))

	require.Len(t, r.events, 3)
	// We're only testing the backtrace here because it's hard to be deterministic
	err := r.events[1]
	result := make(map[string]interface{})
	require.NoError(t, mbson.Unmarshal(err.ToBson(), result))
	backtraceI, ok := result["Backtrace"]
	require.True(t, ok)
	require.NotEmpty(t, backtraceI)
	backtrace, ok := backtraceI.(string)
	require.True(t, ok)
	lines := strings.Split(backtrace, "\n")
	// This vague matching mirrors the Otel-Go behavior here:
	// https://github.com/open-telemetry/opentelemetry-go/blob/248413d6544479f8576a4b68107cb9c78c40f1df/sdk/trace/trace_test.go#L1299-L1300
	require.True(t, strings.HasPrefix(lines[1], "go.opentelemetry.io/otel/sdk/trace.recordStackTrace"))
	require.True(t, strings.HasPrefix(lines[3], "go.opentelemetry.io/otel/sdk/trace.(*recordingSpan).RecordError"))
}

type capturingReporter struct {
	events []reporter.Event
}

func (c *capturingReporter) ReportEvent(e reporter.Event) error {
	c.events = append(c.events, e)
	return nil
}

func (c *capturingReporter) ReportStatus(reporter.Event) error {
	panic("method should not be called")
}

func (c *capturingReporter) Shutdown(context.Context) error {
	return nil
}

func (c *capturingReporter) ShutdownNow() {
	panic("method should not be called")
}

func (c *capturingReporter) Closed() bool {
	return false
}

func (c *capturingReporter) WaitForReady(context.Context) bool {
	return true
}

func (c *capturingReporter) SetServiceKey(string) error {
	panic("method should not be called")
}

func (c *capturingReporter) GetServiceName() string {
	panic("method should not be called")
}

var _ reporter.Reporter = &capturingReporter{}
