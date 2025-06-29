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

package processor

import (
	"context"
	"testing"

	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type recordMock struct {
	span   sdktrace.ReadOnlySpan
	called bool
}

func (r *recordMock) RecordSpan(span sdktrace.ReadOnlySpan) {
	r.span = span
	r.called = true
}

func (r *recordMock) BuildBuiltinMetricsMessage(int32, *metrics.EventQueueStats, *metrics.RateCountSummary, bool) []byte {
	panic("should not be called in this test")
}

func (r *recordMock) BuildCustomMetricsMessage(int32) []byte {
	panic("should not be called in this test")
}

func (r *recordMock) ApmMetricsCap() int32 {
	panic("should not be called in this test")
}

func (r *recordMock) SetApmMetricsCap(int32) {
	panic("should not be called in this test")
}

func (r *recordMock) CustomMetricsCap() int32 {
	panic("should not be called in this test")
}

func (r *recordMock) SetCustomMetricsCap(int32) {
	panic("should not be called in this test")
}

var _ metrics.LegacyRegistry = &recordMock{}

func TestInboundMetricsSpanProcessorOnEnd(t *testing.T) {
	mock := &recordMock{}
	sp := NewInboundMetricsSpanProcessor(mock)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	tracer := tp.Tracer("foo")
	ctx := context.Background()
	_, s := tracer.Start(ctx, "span name")

	// must add entry span
	es, ok := entryspans.Current(s.SpanContext().TraceID())
	require.True(t, ok)
	require.Equal(t, s.SpanContext().SpanID(), es)

	s.End()

	// entry span is removed on span end
	_, ok = entryspans.Current(s.SpanContext().TraceID())
	require.False(t, ok)
	assert.True(t, mock.called)
}

type recordOnlySampler struct{}

func (ro recordOnlySampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	return sdktrace.SamplingResult{
		Decision:   sdktrace.RecordOnly,
		Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
	}
}

func (ro recordOnlySampler) Description() string {
	return "record only sampler"
}

func TestInboundMetricsSpanProcessorOnEndRecordOnly(t *testing.T) {
	mock := &recordMock{}
	sp := NewInboundMetricsSpanProcessor(mock)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
		sdktrace.WithSampler(recordOnlySampler{}),
	)
	tracer := tp.Tracer("foo")
	ctx := context.Background()
	_, s := tracer.Start(ctx, "span name")

	// must add entry span
	es, ok := entryspans.Current(s.SpanContext().TraceID())
	require.True(t, ok)
	require.Equal(t, s.SpanContext().SpanID(), es)

	s.End()

	// MUST remove entry span; because it's NOT sampled, exporter will NOT handle deletion
	es, ok = entryspans.Current(s.SpanContext().TraceID())
	require.False(t, ok)
	require.False(t, es.IsValid())
	assert.True(t, mock.called)
}

func TestInboundMetricsSpanProcessorOnEndWithLocalParent(t *testing.T) {
	mock := &recordMock{}
	sp := NewInboundMetricsSpanProcessor(mock)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sp))
	tracer := tp.Tracer("foo")
	ctx, s1 := tracer.Start(context.Background(), "span name")

	// must add entry span
	es, ok := entryspans.Current(s1.SpanContext().TraceID())
	require.True(t, ok)
	require.Equal(t, s1.SpanContext().SpanID(), es)

	_, s2 := tracer.Start(ctx, "child span")
	// s2 is *not* an entry span, so s1 should remain the current entry span
	es, ok = entryspans.Current(s1.SpanContext().TraceID())
	require.True(t, ok)
	require.Equal(t, s1.SpanContext().SpanID(), es)

	s2.End()

	assert.False(t, mock.called)
}

func TestInboundMetricsSpanProcessorOnEndWithRemoteParent(t *testing.T) {
	mock := &recordMock{}
	sp := NewInboundMetricsSpanProcessor(mock)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sp))
	tracer := tp.Tracer("foo")
	ctx := context.Background()
	ctx, s := tracer.Start(ctx, "span name")
	ctx = trace.ContextWithRemoteSpanContext(ctx, s.SpanContext())
	_, s2 := tracer.Start(ctx, "child span")
	s2.End()

	assert.True(t, mock.called)
}

func TestSpanIsRemovedFromStateManagerAfterSpanEnds(t *testing.T) {
	ctx := context.Background()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(NewInboundMetricsSpanProcessor(&recordMock{})))
	tracer := tp.Tracer("foo")
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			require.NoError(t, err)
		}
	}()

	_, entrySpan := tracer.Start(ctx, "entry-span")

	sid, ok := entryspans.Current(entrySpan.SpanContext().TraceID())
	require.True(t, ok)
	require.Equal(t, entrySpan.SpanContext().SpanID(), sid)

	entrySpan.End()

	_, ok = entryspans.Current(entrySpan.SpanContext().TraceID())
	require.False(t, ok)
}
