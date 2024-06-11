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
	"github.com/solarwinds/apm-go/internal/testutils"
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
	span3 = trace.SpanID{0x3}
	span4 = trace.SpanID{0x4}
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

func TestCurrent(t *testing.T) {
	state := state.(*stdManager)
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
	require.NoError(t, Push(span.(sdktrace.ReadOnlySpan)))
	require.Equal(t,
		[]*entrySpan{
			{spanId: span.SpanContext().SpanID()},
		},
		state.spans[span.SpanContext().TraceID()],
	)

	var nonEntrySpan trace.Span
	_, nonEntrySpan = tr.Start(ctx, "B")
	err = Push(nonEntrySpan.(sdktrace.ReadOnlySpan))
	require.Error(t, err)
	require.Equal(t, NotEntrySpan, err)
}

func TestSetTransactionName(t *testing.T) {
	// reset state
	state = &stdManager{spans: make(map[trace.TraceID][]*entrySpan)}
	state := state.(*stdManager)

	err := SetTransactionName(traceA, "foo bar")
	require.Error(t, err)

	state.push(traceA, span1)
	state.push(traceA, span2)

	err = SetTransactionName(traceA, "foo bar")
	require.Equal(t,
		[]*entrySpan{
			{spanId: span1},
			{spanId: span2, txnName: "foo bar"},
		},
		state.spans[traceA],
	)

	require.NoError(t, err)
	curr, ok := state.current(traceA)
	require.True(t, ok)
	require.Equal(t, span2, curr.spanId)
	require.Equal(t, "foo bar", curr.txnName)

	require.Equal(t, "foo bar", GetTransactionName(traceA))
	require.Equal(t, "", GetTransactionName(traceB))

	sid, ok := state.pop(traceA)
	require.True(t, ok)
	require.Equal(t, span2, sid)

	require.Equal(t, "", GetTransactionName(traceA))
	require.Equal(t, "", GetTransactionName(traceB))

	err = SetTransactionName(traceA, "another")
	require.NoError(t, err)
	require.Equal(t, "another", GetTransactionName(traceA))
	require.Equal(t, "", GetTransactionName(traceB))
}

func TestDelete(t *testing.T) {
	// reset state
	state = &stdManager{spans: make(map[trace.TraceID][]*entrySpan)}
	state := state.(*stdManager)

	err := state.delete(traceA, span1)
	require.Error(t, err)
	require.Equal(t, "could not find trace id", err.Error())

	state.push(traceA, span1)
	state.push(traceA, span2)
	state.push(traceA, span3)

	err = state.delete(traceA, span4)
	require.Error(t, err)
	require.Equal(t, "could not find span id", err.Error())

	err = state.delete(traceA, span2)
	require.NoError(t, err)
	require.Equal(t,
		[]*entrySpan{
			{spanId: span1},
			{spanId: span3},
		},
		state.spans[traceA],
	)

	tr, teardown := testutils.TracerSetup()
	defer teardown()
	_, s := tr.Start(context.Background(), "foo bar baz")
	state.push(s.SpanContext().TraceID(), s.SpanContext().SpanID())
	require.Equal(t,
		[]*entrySpan{
			{spanId: s.SpanContext().SpanID()},
		},
		state.spans[s.SpanContext().TraceID()],
	)
	err = Delete(s.(sdktrace.ReadOnlySpan))
	require.NoError(t, err)
	_, ok := state.spans[s.SpanContext().TraceID()]
	require.False(t, ok)
}
