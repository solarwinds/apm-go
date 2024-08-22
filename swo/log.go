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
	"context"
	"fmt"
	"github.com/solarwinds/apm-go/internal/state"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

type LoggableTraceContext struct {
	TraceID     trace.TraceID    `json:"trace_id,omitempty"`
	SpanID      trace.SpanID     `json:"span_id,omitempty"`
	TraceFlags  trace.TraceFlags `json:"trace_flags,omitempty"`
	ServiceName string           `json:"service_name,omitempty"`
}

// String returns a string representation that is usable in a log
// Example: trace_id=d4261c67357f99f39958b14f99da7e6c span_id=1280450002ba77b3 trace_flags=01 resource.service.name=my-service
func (l LoggableTraceContext) String() string {
	return fmt.Sprintf(
		"trace_id=%s span_id=%s trace_flags=%s resource.service.name=%s",
		l.TraceID,
		l.SpanID,
		l.TraceFlags,
		l.ServiceName,
	)
}

// IsValid returns true if both TraceID and SpanID are valid
func (l LoggableTraceContext) IsValid() bool {
	return l.TraceID.IsValid() && l.SpanID.IsValid()
}

// LoggableTrace returns a LoggableTraceContext from a given context.Context and the configured service name
func LoggableTrace(ctx context.Context) LoggableTraceContext {
	return LoggableTraceFromSpanContext(trace.SpanContextFromContext(ctx))
}

// LoggableTraceFromSpanContext returns a LoggableTraceContext from a given SpanContext and the configured service name
func LoggableTraceFromSpanContext(ctx trace.SpanContext) LoggableTraceContext {
	return LoggableTraceContext{
		TraceID:     ctx.TraceID(),
		SpanID:      ctx.SpanID(),
		TraceFlags:  ctx.TraceFlags(),
		ServiceName: state.GetServiceName(),
	}
}

// LogHandler is a custom slog handler that adds a trace ID from the trace context to each log entry
type LogHandler struct {
	wrapped slog.Handler
}

var _ slog.Handler = &LogHandler{}

// NewLogHandler creates a new LogHandler
func NewLogHandler(wrapped slog.Handler) *LogHandler {
	return &LogHandler{wrapped: wrapped}
}

// Enabled calls slog Enabled
func (h *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.wrapped.Enabled(ctx, level)
}

// Handle adds trace context to the record, in the format that allows SWO to
// associate log lines with traces
func (h *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	traceContext := LoggableTrace(ctx)
	if traceContext.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", traceContext.TraceID.String()),
			slog.String("span_id", traceContext.SpanID.String()),
			slog.String("trace_flags", traceContext.TraceFlags.String()),
		)
	}
	record.AddAttrs(
		slog.String("resource.service.name", traceContext.ServiceName),
	)
	return h.wrapped.Handle(ctx, record)
}

// WithAttrs calls slog WithAttrs
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{wrapped: h.wrapped.WithAttrs(attrs)}
}

// WithGroup calls slog WithGroup
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{wrapped: h.wrapped.WithGroup(name)}
}
