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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/w3cfmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type sampler struct {
}

func NewSampler() sdktrace.Sampler {
	return sampler{}
}

var _ sdktrace.Sampler = sampler{}

func (s sampler) Description() string {
	return "SolarWinds APM Sampler"
}

var alwaysSampler = sdktrace.AlwaysSample()
var neverSampler = sdktrace.NeverSample()

func hydrateTraceState(psc trace.SpanContext, xto xtrace.Options) trace.TraceState {
	var ts trace.TraceState
	if !psc.IsValid() {
		// create new tracestate
		ts = trace.TraceState{}
	} else {
		ts = psc.TraceState()
	}
	if xto.Opts() != "" {
		// TODO: NH-5731
	}
	return ts
}

func (s sampler) ShouldSample(params sdktrace.SamplingParameters) sdktrace.SamplingResult {
	psc := trace.SpanContextFromContext(params.ParentContext)

	var result sdktrace.SamplingResult
	if psc.IsValid() && !psc.IsRemote() {
		if psc.IsSampled() {
			result = alwaysSampler.ShouldSample(params)
		} else {
			result = neverSampler.ShouldSample(params)
		}
	} else {
		// TODO url
		url := ""
		xto := xtrace.GetXTraceOptions(params.ParentContext)
		ttMode := getTtMode(xto)
		// If parent context is not valid, swState will also not be valid
		swState := w3cfmt.GetSwTraceState(psc)
		traceDecision := reporter.ShouldTraceRequestWithURL(
			params.Name, // "may not be super relevant" -- lin
			swState.IsValid(),
			url,
			ttMode,
			&swState,
		)
		var decision sdktrace.SamplingDecision
		if traceDecision.Trace() {
			decision = sdktrace.RecordAndSample
		} else {
			decision = sdktrace.RecordOnly
		}
		ts := hydrateTraceState(psc, xto)
		result = sdktrace.SamplingResult{
			Decision:   decision,
			Tracestate: ts,
		}
	}

	// TODO NH-43627
	return result

}

func getTtMode(xto xtrace.Options) reporter.TriggerTraceMode {
	if xto.TriggerTrace() {
		switch xto.SignatureState() {
		case xtrace.ValidSignature:
			return reporter.ModeRelaxedTriggerTrace
		case xtrace.InvalidSignature:
			return reporter.ModeInvalidTriggerTrace
		default:
			return reporter.ModeStrictTriggerTrace
		}
	} else {
		return reporter.ModeTriggerTraceNotPresent
	}
}
