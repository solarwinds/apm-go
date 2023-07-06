package solarwinds_apm

import (
	"context"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

type recordMock struct {
	span        sdktrace.ReadOnlySpan
	isAppoptics bool
	called      bool
}

func TestSolarWindsInboundMetricsSpanProcessorOnEnd(t *testing.T) {
	mock := &recordMock{}
	recordFunc = func(span sdktrace.ReadOnlySpan, isAppoptics bool) {
		mock.span = span
		mock.isAppoptics = isAppoptics
		mock.called = true
	}
	defer func() {
		recordFunc = metrics.RecordSpan
	}()
	sp := &SolarWindsInboundMetricsSpanProcessor{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sp))
	tracer := tp.Tracer("foo")
	ctx := context.Background()
	_, s := tracer.Start(ctx, "span name")
	s.End()

	assert.True(t, mock.called)
	assert.False(t, mock.isAppoptics)
}

func TestSolarWindsInboundMetricsSpanProcessorOnEndWithLocalParent(t *testing.T) {
	mock := &recordMock{}
	recordFunc = func(span sdktrace.ReadOnlySpan, isAppoptics bool) {
		mock.span = span
		mock.isAppoptics = isAppoptics
		mock.called = true
	}
	defer func() {
		recordFunc = metrics.RecordSpan
	}()
	sp := &SolarWindsInboundMetricsSpanProcessor{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sp))
	tracer := tp.Tracer("foo")
	ctx := context.Background()
	ctx, _ = tracer.Start(ctx, "span name")
	_, s2 := tracer.Start(ctx, "child span")
	s2.End()

	assert.False(t, mock.called)
}

func TestSolarWindsInboundMetricsSpanProcessorOnEndWithRemoteParent(t *testing.T) {
	mock := &recordMock{}
	recordFunc = func(span sdktrace.ReadOnlySpan, isAppoptics bool) {
		mock.span = span
		mock.isAppoptics = isAppoptics
		mock.called = true
	}
	defer func() {
		recordFunc = metrics.RecordSpan
	}()
	sp := &SolarWindsInboundMetricsSpanProcessor{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sp))
	tracer := tp.Tracer("foo")
	ctx := context.Background()
	ctx, s := tracer.Start(ctx, "span name")
	ctx = trace.ContextWithRemoteSpanContext(ctx, s.SpanContext())
	_, s2 := tracer.Start(ctx, "child span")
	s2.End()

	assert.True(t, mock.called)
}
