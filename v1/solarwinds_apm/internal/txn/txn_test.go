package txn

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"sync"
	"testing"
)

var (
	spanId  = trace.SpanID{0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1}
	spanId2 = trace.SpanID{0x2, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1}
	traceId = trace.TraceID{0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}
	ctx     = trace.NewSpanContext(trace.SpanContextConfig{SpanID: spanId, TraceID: traceId})
	ctx2    = trace.NewSpanContext(trace.SpanContextConfig{SpanID: spanId2, TraceID: traceId})
)

func TestMakeKey(t *testing.T) {
	key := makeKey(ctx)

	require.Equal(t, spanId[:], key[:8])
	require.Equal(t, traceId[:], key[8:])
	require.Len(t, key, 24)

	key = makeKey(ctx2)

	require.Equal(t, spanId2[:], key[:8])
	require.Equal(t, traceId[:], key[8:])
	require.Len(t, key, 24)
}

func TestSetGetRemove(t *testing.T) {
	require.Len(t, txns, 0)
	name, ok := Get(ctx)
	require.False(t, ok)
	require.Equal(t, "", name)

	ok = Remove(ctx)
	require.False(t, ok)

	Set(ctx, "foobar")
	require.Len(t, txns, 1)

	name, ok = Get(ctx)
	require.True(t, ok)
	require.Equal(t, "foobar", name)

	Set(ctx2, "other txn")
	require.Len(t, txns, 2)
	name, ok = Get(ctx2)
	require.True(t, ok)
	require.Equal(t, "other txn", name)

	ok = Remove(ctx2)
	require.True(t, ok)
	require.Len(t, txns, 1)

	name, ok = Get(ctx2)
	require.False(t, ok)
	require.Equal(t, "", name)

	name, ok = Get(ctx)
	require.True(t, ok)
	require.Equal(t, "foobar", name)

	Set(ctx, "overwrite")
	require.Len(t, txns, 1)

	name, ok = Get(ctx)
	require.True(t, ok)
	require.Equal(t, "overwrite", name)

	ok = Remove(ctx)
	require.Len(t, txns, 0)
	require.True(t, ok)
	ok = Remove(ctx)
	require.False(t, ok)
	ok = Remove(ctx2)
	require.False(t, ok)
}

func TestConcurrentSafety(t *testing.T) {
	// Reasonably good effort to cause a deadlock or panic
	numGoRoutines := 1000
	var countdown sync.WaitGroup
	countdown.Add(numGoRoutines)
	var start sync.WaitGroup
	start.Add(1)

	for i := 0; i < numGoRoutines; i++ {
		go func() {
			start.Wait()
			for i := 0; i < 1000; i++ {
				uid, err := uuid.NewRandom()
				require.NoError(t, err)
				Set(ctx, uid.String())
				Get(ctx)
				Remove(ctx2)
				Set(ctx2, uid.String())
				Get(ctx2)
				Remove(ctx2)
			}
			countdown.Done()
		}()
	}
	start.Done()
	countdown.Wait()

}
