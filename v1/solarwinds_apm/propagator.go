package solarwinds_apm

import (
	"context"
	"fmt"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/w3cfmt"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TODO rename
	cTraceState = "tracestate"
	traceParent = "traceparent"
	VendorID    = "sw"
)

type SolarwindsPropagator struct{}

var _ propagation.TextMapPropagator = SolarwindsPropagator{}

func (swp SolarwindsPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	fmt.Println("SolarwindsPropagator Inject called")
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	fmt.Println("span, ctx", span, sc)
	swVal := w3cfmt.SwFromCtx(sc)
	traceStateHeader := carrier.Get(cTraceState)

	var traceState trace.TraceState
	if traceStateHeader == "" {
		if !sc.IsValid() {
			return
		}
		// Create new trace state
		traceState = trace.TraceState{}
	} else {
		var err error
		traceState, err = trace.ParseTraceState(traceStateHeader)
		if err != nil {
			fmt.Println("err parsing trace state from string!!!")
			return
		}
	}
	fmt.Println("Inserting tracestate", swVal)
	// Note: Insert will update the key if it exists
	traceState.Insert(VendorID, swVal)

	// TODO maybe. From the python apm library: Remove any
	// xtrace_options_response stored for ResponsePropagator
	carrier.Set(cTraceState, traceState.String())
}

func (swp SolarwindsPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return ctx
}

// Fields returns the keys who's values are set with Inject.
func (swp SolarwindsPropagator) Fields() []string {
	return []string{cTraceState}
}
