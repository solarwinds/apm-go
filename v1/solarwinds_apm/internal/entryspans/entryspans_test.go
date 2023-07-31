package entryspans

import (
	"context"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/testutils"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

var (
	traceA = trace.TraceID{0xA}
	traceB = trace.TraceID{0xB}

	span1 = trace.SpanID{0x1}
	span2 = trace.SpanID{0x2}
)

func TestCurrent(t *testing.T) {
	sid, ok := Current(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	sid, ok = Current(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	state.push(traceA, span1)

	sid, ok = Current(traceA)
	require.True(t, ok)
	require.Equal(t, span1, sid)

	sid, ok = Current(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	state.push(traceA, span2)

	sid, ok = Current(traceA)
	require.True(t, ok)
	require.Equal(t, span2, sid)

	sid, ok = Current(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	sid, ok = state.pop(traceA)
	require.True(t, ok)
	require.Equal(t, span2, sid)

	sid, ok = Current(traceA)
	require.True(t, ok)
	require.Equal(t, span1, sid)

	sid, ok = state.pop(traceA)
	require.True(t, ok)
	require.Equal(t, span1, sid)

	sid, ok = Current(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	// this is an invalid state, but we handle it
	state.spans[traceA] = []trace.SpanID{}
	sid, ok = Current(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())
}

func TestPush(t *testing.T) {
	var err error
	tr, teardown := testutils.TracerSetup()
	defer teardown()

	ctx := context.Background()
	var span trace.Span
	ctx, span = tr.Start(ctx, "A")
	require.NoError(t, Push(span.(sdktrace.ReadOnlySpan)))
	require.Equal(t,
		[]trace.SpanID{
			span.SpanContext().SpanID(),
		},
		state.spans[span.SpanContext().TraceID()],
	)

	var nonEntrySpan trace.Span
	ctx, nonEntrySpan = tr.Start(ctx, "B")
	err = Push(nonEntrySpan.(sdktrace.ReadOnlySpan))
	require.Error(t, err)
	require.Equal(t, NotEntrySpan, err)
}

func TestPop(t *testing.T) {
	sid, ok := Pop(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	sid, ok = Pop(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	state.push(traceA, span1)
	state.push(traceA, span2)

	sid, ok = Pop(traceA)
	require.Equal(t, span2, sid)
	require.True(t, ok)

	sid, ok = Pop(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	sid, ok = Pop(traceA)
	require.Equal(t, span1, sid)
	require.True(t, ok)

	sid, ok = Pop(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	// this is an invalid state, but we handle it
	state.spans[traceA] = []trace.SpanID{}
	sid, ok = Pop(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())
	// this should be cleaned up
	_, ok = state.spans[traceA]
	require.False(t, ok)
}
