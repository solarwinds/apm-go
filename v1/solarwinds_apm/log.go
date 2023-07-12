package solarwinds_apm

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/trace"
)

// LoggableTraceIDFromContext Returns a loggable trace ID from the given
// context.Context for log injection, or an empty string if the trace
// is invalid
func LoggableTraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	return LoggableTraceIDFromSpanContext(sc)
}

func LoggableTraceIDFromSpanContext(ctx trace.SpanContext) string {
	if !ctx.IsValid() {
		return ""
	}
	tid := ctx.TraceID().String()
	var sampled string
	if ctx.IsSampled() {
		sampled = "1"
	} else {
		sampled = "0"
	}
	return fmt.Sprintf("%s-%s", tid, sampled)
}
