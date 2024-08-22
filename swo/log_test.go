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

package swo

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/solarwinds/apm-go/internal/state"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"testing"
	"time"
)

func TestLoggableTraceIDFromContext(t *testing.T) {
	prev := state.GetServiceName()
	state.SetServiceName("test-service")
	defer state.SetServiceName(prev)
	ctx := context.Background()
	lt := LoggableTrace(ctx)
	require.Equal(t, LoggableTraceContext{
		TraceID:     trace.TraceID{},
		SpanID:      trace.SpanID{},
		TraceFlags:  0,
		ServiceName: "test-service",
	}, lt)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x22},
		SpanID:     trace.SpanID{0x11},
		TraceFlags: trace.FlagsSampled,
	})
	require.False(t, lt.IsValid())
	require.Equal(t,
		"trace_id=00000000000000000000000000000000 span_id=0000000000000000 trace_flags=00 resource.service.name=test-service",
		lt.String())

	ctx = trace.ContextWithSpanContext(ctx, sc)
	lt = LoggableTrace(ctx)
	require.Equal(t, LoggableTraceContext{
		TraceID:     sc.TraceID(),
		SpanID:      sc.SpanID(),
		TraceFlags:  sc.TraceFlags(),
		ServiceName: "test-service",
	}, lt)
	require.True(t, lt.IsValid())
	require.Equal(t,
		"trace_id=22000000000000000000000000000000 span_id=1100000000000000 trace_flags=01 resource.service.name=test-service",
		lt.String())

	sc = trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x33},
		SpanID:     trace.SpanID{0xAA},
		TraceFlags: trace.FlagsSampled,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)
	lt = LoggableTrace(ctx)
	require.Equal(t, LoggableTraceContext{
		TraceID:     sc.TraceID(),
		SpanID:      sc.SpanID(),
		TraceFlags:  sc.TraceFlags(),
		ServiceName: "test-service",
	}, lt)
	require.True(t, lt.IsValid())
	require.Equal(t,
		"trace_id=33000000000000000000000000000000 span_id=aa00000000000000 trace_flags=01 resource.service.name=test-service",
		lt.String())
}

func TestHandle(t *testing.T) {
	prevServiceName := state.GetServiceName()
	defer state.SetServiceName(prevServiceName)
	serviceName := "test-service"
	state.SetServiceName(serviceName)

	type logLine struct {
		Time        time.Time `json:"time"`
		Level       string    `json:"level"`
		Msg         string    `json:"msg"`
		ServiceName string    `json:"resource.service.name"`
		TraceID     string    `json:"trace_id"`
		SpanID      string    `json:"span_id"`
		TraceFlags  string    `json:"trace_flags"`
	}

	tests := []struct {
		name     string
		traceCtx LoggableTraceContext
	}{
		{"info level with trace", LoggableTraceContext{
			TraceID:    trace.TraceID{0x11},
			SpanID:     trace.SpanID{0x22},
			TraceFlags: trace.FlagsSampled,
		}},
		{"empty trace", LoggableTraceContext{}},
	}

	now := time.Now().Truncate(time.Second).UTC()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			writer := bytes.NewBuffer(make([]byte, 0))
			baseHandler := slog.NewJSONHandler(writer, &slog.HandlerOptions{})
			handler := NewLogHandler(baseHandler)

			spanContext := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    tt.traceCtx.TraceID,
				SpanID:     tt.traceCtx.SpanID,
				TraceFlags: tt.traceCtx.TraceFlags,
			})
			ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
			record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

			// act
			err := handler.Handle(ctx, record)
			require.NoError(t, err)

			// assert
			logOutput := writer.String()

			var line logLine
			err = json.Unmarshal([]byte(logOutput), &line)
			require.NoError(t, err)

			if tt.traceCtx.IsValid() {
				require.Equal(t, tt.traceCtx.TraceID.String(), line.TraceID)
				require.Equal(t, tt.traceCtx.SpanID.String(), line.SpanID)
				require.Equal(t, tt.traceCtx.TraceFlags.String(), line.TraceFlags)
			} else {
				require.Empty(t, line.TraceID)
				require.Empty(t, line.SpanID)
				require.Empty(t, line.TraceFlags)
			}
			require.Equal(t, "test message", line.Msg)
			require.Equal(t, serviceName, line.ServiceName)
			require.Equal(t, now, line.Time)

		})
	}
}
