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

package entryspans

import (
	"context"
	"testing"

	"github.com/solarwinds/apm-go/internal/testutils"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	traceA = trace.TraceID{0xA}
	traceB = trace.TraceID{0xB}
)

func (e *stdManager) pop(tid trace.TraceID) (trace.SpanID, bool) {
	e.mut.Lock()
	defer e.mut.Unlock()

	if list, ok := e.spans[tid]; ok {
		l := len(list)
		if l == 0 {
			delete(e.spans, tid)
			return nullSpanID, false
		} else if l == 1 {
			delete(e.spans, tid)
			return list[0].spanId, true
		} else {
			item := list[l-1]
			list = list[:l-1]
			e.spans[tid] = list
			return item.spanId, true
		}
	} else {
		return nullSpanID, ok
	}
}

func (e *stdManager) reset() {
	e.mut.Lock()
	defer e.mut.Unlock()
	clear(e.spans)
}

func TestCurrent(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()

	ctx := context.Background()
	var span1, span2 trace.Span
	_, span1 = tr.Start(ctx, "A")
	_, span2 = tr.Start(ctx, "B")

	state := state.(*stdManager)
	sid, ok := Current(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	sid, ok = Current(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	state.push(traceA, span1.(sdktrace.ReadWriteSpan))

	sid, ok = Current(traceA)
	require.True(t, ok)
	require.Equal(t, span1.SpanContext().SpanID(), sid)

	sid, ok = Current(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	state.push(traceA, span2.(sdktrace.ReadWriteSpan))

	sid, ok = Current(traceA)
	require.True(t, ok)
	require.Equal(t, span2.SpanContext().SpanID(), sid)

	sid, ok = Current(traceB)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	sid, ok = state.pop(traceA)
	require.True(t, ok)
	require.Equal(t, span2.SpanContext().SpanID(), sid)

	sid, ok = Current(traceA)
	require.True(t, ok)
	require.Equal(t, span1.SpanContext().SpanID(), sid)

	sid, ok = state.pop(traceA)
	require.True(t, ok)
	require.Equal(t, span1.SpanContext().SpanID(), sid)

	sid, ok = Current(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())

	// this is an invalid state, but we handle it
	state.spans[traceA] = []*entrySpan{}
	sid, ok = Current(traceA)
	require.False(t, ok)
	require.False(t, sid.IsValid())
}

func TestPush(t *testing.T) {
	state := state.(*stdManager)
	var err error
	tr, teardown := testutils.TracerSetup()
	defer teardown()

	ctx := context.Background()
	var span trace.Span
	ctx, span = tr.Start(ctx, "A")
	require.NoError(t, Push(span.(sdktrace.ReadWriteSpan)))
	require.Equal(t,
		[]*entrySpan{
			{spanId: span.SpanContext().SpanID(), spanHandle: span.(sdktrace.ReadWriteSpan)},
		},
		state.spans[span.SpanContext().TraceID()],
	)

	var nonEntrySpan trace.Span
	_, nonEntrySpan = tr.Start(ctx, "B")
	err = Push(nonEntrySpan.(sdktrace.ReadWriteSpan))
	require.Error(t, err)
	require.Equal(t, NotEntrySpan, err)
}

func TestSetTransactionName(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()

	ctx := context.Background()
	var span1, span2 trace.Span
	_, span1 = tr.Start(ctx, "A")
	_, span2 = tr.Start(ctx, "B")

	state := state.(*stdManager)
	state.reset()

	err := SetTransactionName(traceA, "foo bar")
	require.Error(t, err)

	state.push(traceA, span1.(sdktrace.ReadWriteSpan))
	state.push(traceA, span2.(sdktrace.ReadWriteSpan))

	err = SetTransactionName(traceA, "foo bar")
	require.Equal(t,
		[]*entrySpan{
			{spanHandle: span1.(sdktrace.ReadWriteSpan), spanId: span1.SpanContext().SpanID()},
			{spanHandle: span2.(sdktrace.ReadWriteSpan), spanId: span2.SpanContext().SpanID(), txnName: "foo bar"},
		},
		state.spans[traceA],
	)

	require.NoError(t, err)
	curr, ok := state.current(traceA)
	require.True(t, ok)
	require.Equal(t, span2.SpanContext().SpanID(), curr.spanId)
	require.Equal(t, "foo bar", curr.txnName)

	require.Equal(t, "foo bar", GetTransactionName(traceA))
	require.Equal(t, "", GetTransactionName(traceB))

	sid, ok := state.pop(traceA)
	require.True(t, ok)
	require.Equal(t, span2.SpanContext().SpanID(), sid)

	require.Equal(t, "", GetTransactionName(traceA))
	require.Equal(t, "", GetTransactionName(traceB))

	err = SetTransactionName(traceA, "another")
	require.NoError(t, err)
	require.Equal(t, "another", GetTransactionName(traceA))
	require.Equal(t, "", GetTransactionName(traceB))
}

func TestDelete(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()

	ctx := context.Background()
	var span1, span2, span3, span4 trace.Span
	_, span1 = tr.Start(ctx, "A")
	_, span2 = tr.Start(ctx, "B")
	_, span3 = tr.Start(ctx, "C")
	_, span4 = tr.Start(ctx, "D")

	state := state.(*stdManager)
	state.reset()

	err := state.delete(traceA, span1.SpanContext().SpanID())
	require.Error(t, err)
	require.Equal(t, "could not find trace id", err.Error())

	state.push(traceA, span1.(sdktrace.ReadWriteSpan))
	state.push(traceA, span2.(sdktrace.ReadWriteSpan))
	state.push(traceA, span3.(sdktrace.ReadWriteSpan))

	err = state.delete(traceA, span4.SpanContext().SpanID())
	require.Error(t, err)
	require.Equal(t, "could not find span id", err.Error())

	err = state.delete(traceA, span2.SpanContext().SpanID())
	require.NoError(t, err)
	require.Equal(t,
		[]*entrySpan{
			{spanHandle: span1.(sdktrace.ReadWriteSpan), spanId: span1.SpanContext().SpanID()},
			{spanHandle: span3.(sdktrace.ReadWriteSpan), spanId: span3.SpanContext().SpanID()},
		},
		state.spans[traceA],
	)

	_, s := tr.Start(context.Background(), "foo bar baz")
	state.push(s.SpanContext().TraceID(), s.(sdktrace.ReadWriteSpan))
	require.Equal(t,
		[]*entrySpan{
			{spanHandle: s.(sdktrace.ReadWriteSpan), spanId: s.SpanContext().SpanID()},
		},
		state.spans[s.SpanContext().TraceID()],
	)
	err = Delete(s.(sdktrace.ReadOnlySpan))
	require.NoError(t, err)
	_, ok := state.spans[s.SpanContext().TraceID()]
	require.False(t, ok)
}
