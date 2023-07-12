package solarwinds_apm

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestLoggableTraceIDFromContext(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, "", LoggableTraceIDFromContext(ctx))
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x22},
		SpanID:  trace.SpanID{0x11},
	})

	ctx = trace.ContextWithSpanContext(ctx, sc)
	require.Equal(t, "22000000000000000000000000000000-0", LoggableTraceIDFromContext(ctx))

	sc = trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x33},
		SpanID:     trace.SpanID{0xAA},
		TraceFlags: trace.FlagsSampled,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)
	require.Equal(t, "33000000000000000000000000000000-1", LoggableTraceIDFromContext(ctx))
}
