package solarwinds_apm

import (
	"context"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"testing"
)

type recordMock struct {
	span        metrics.RoSpan
	isAppoptics bool
	called      bool
}

func TestSolarWindsInboundMetricsSpanProcessorOnEnd(t *testing.T) {
	mock := &recordMock{}
	recordFunc = func(span metrics.RoSpan, isAppoptics bool) {
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
	s.End()

	assert.True(t, mock.called)
	assert.False(t, mock.isAppoptics)
}

func TestSolarWindsInboundMetricsSpanProcessorOnEndWithParent(t *testing.T) {
	mock := &recordMock{}
	recordFunc = func(span metrics.RoSpan, isAppoptics bool) {
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
	ctx, s2 := tracer.Start(ctx, "child span")
	s2.End()

	assert.False(t, mock.called)
}
