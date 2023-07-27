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

package solarwinds_apm

import (
	"context"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/constants"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/swotel"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/w3cfmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type SolarwindsPropagator struct{}

var _ propagation.TextMapPropagator = SolarwindsPropagator{}

func (swp SolarwindsPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	swVal := w3cfmt.SwFromCtx(sc)
	traceStateHeader := carrier.Get(constants.TraceState)

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
	traceState, err = swotel.SetSw(traceState, swVal)
	if err != nil {
		log.Debugf("could not insert vendor info into tracestate `%s`", swVal)
		return
	}

	traceState, err = swotel.RemoveInternalState(traceState, swotel.XTraceOptResp)
	if err != nil {
		log.Debugf("could not remove xtrace options resp from trace state", err)
	}
	carrier.Set(constants.TraceState, traceState.String())
}

func (swp SolarwindsPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	xtraceOptionsHeader := carrier.Get(xtrace.OptionsHeaderName)
	if xtraceOptionsHeader != "" {
		ctx = context.WithValue(ctx, xtrace.OptionsKey, xtraceOptionsHeader)
	}

	xtraceSig := carrier.Get(xtrace.OptionsSigHeaderName)
	if xtraceSig != "" {
		ctx = context.WithValue(ctx, xtrace.SignatureKey, xtraceSig)
	}

	return ctx
}

// Fields returns the keys who's values are set with Inject.
func (swp SolarwindsPropagator) Fields() []string {
	return []string{constants.TraceState}
}
