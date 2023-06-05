package solarwinds_apm

import (
	"context"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"

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
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	swVal := w3cfmt.SwFromCtx(sc)
	traceStateHeader := carrier.Get(cTraceState)

	var traceState trace.TraceState
	var err error
	if traceStateHeader == "" {
		if !sc.IsValid() {
			return
		}
		// Create new trace state
		traceState = trace.TraceState{}
	} else {
		traceState, err = trace.ParseTraceState(traceStateHeader)
		if err != nil {
			log.Debugf("error parsing trace state `%s`", traceStateHeader)
			return
		}
	}
	// Note: Insert will update the key if it exists
	traceState, err = traceState.Insert(VendorID, swVal)
	if err != nil {
		log.Debugf("could not insert vendor info into tracestate `%s`", swVal)
		return
	}

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
