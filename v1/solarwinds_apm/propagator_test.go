package solarwinds_apm

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const spanIdHex = "0123456789abcdef"
const traceIdHex = "44444444444444443333333333333333"

var spanId, err1 = trace.SpanIDFromHex(spanIdHex)
var traceId, err2 = trace.TraceIDFromHex(traceIdHex)

func init() {
	for _, err := range []error{err1, err2} {
		if err != nil {
			log.Fatal("Fatal error: ", err)
		}
	}
}

func TestInjectInvalidSpanContext(t *testing.T) {
	sc := trace.SpanContext{}
	assert.False(t, sc.IsValid())
	carrier := propagation.MapCarrier{}
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, "", carrier.Get("tracestate"))
}

func TestInjectNoTracestate(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: 0,
	})
	carrier := propagation.MapCarrier{}
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, fmt.Sprintf("sw=%s-00", spanIdHex), carrier.Get("tracestate"))
}

func TestInjectWithTracestateNoSw(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: 0,
	})
	carrier := propagation.MapCarrier{}
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, fmt.Sprintf("sw=%s-00", spanIdHex), carrier.Get("tracestate"))
}

func TestInjectWithTracestatePrevSw(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: 0,
	})
	carrier := propagation.MapCarrier{}
	carrier.Set("tracestate", "sw=012301230-00")
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, fmt.Sprintf("sw=%s-00", spanIdHex), carrier.Get("tracestate"))
}

func TestExtract(t *testing.T) {
	ctx := context.TODO()
	carrier := propagation.MapCarrier{}

	p := SolarwindsPropagator{}
	newCtx := p.Extract(ctx, carrier)
	assert.Equal(t, ctx, newCtx)
}

func TestFields(t *testing.T) {
	p := SolarwindsPropagator{}
	assert.Equal(t, []string{"tracestate"}, p.Fields())
}
