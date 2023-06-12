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
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/w3cfmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type sampler struct {
	decider
}

func NewSampler() sdktrace.Sampler {
	return sampler{defaultDecider}
}

type decider interface {
	ShouldTraceRequestWithURL(layer string, traced bool, url string, ttMode reporter.TriggerTraceMode) reporter.SampleDecision
}

type reporterDecider struct{}

func (r reporterDecider) ShouldTraceRequestWithURL(layer string, traced bool, url string, ttMode reporter.TriggerTraceMode) reporter.SampleDecision {
	return reporter.ShouldTraceRequestWithURL(layer, traced, url, ttMode)
}

var defaultDecider decider = reporterDecider{}

var _ sdktrace.Sampler = sampler{}

func (s sampler) Description() string {
	return "SolarWinds APM Sampler"
}

var alwaysSampler = sdktrace.AlwaysSample()
var neverSampler = sdktrace.NeverSample()

func (s sampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	parentContext := parameters.ParentContext
	traced := false

	psc := trace.SpanContextFromContext(parentContext)
	fmt.Println("psc", psc)

	// If parent context is not valid, swState will also not be valid
	swState := w3cfmt.GetSwTraceState(psc)
	xto := xtrace.GetXTraceOptions(parentContext)
	log.Debug("XTO", xto)
	fmt.Println("XTO", xto)

	var result sdktrace.SamplingResult

	if swState.IsValid() && !psc.IsRemote() {
		// Follow upstream trace decision
		if swState.Flags().IsSampled() {
			result = alwaysSampler.ShouldSample(parameters)
		} else {
			result = neverSampler.ShouldSample(parameters)
		}
	} else {
		// Broken or non-existent tracestate; treat as a new trace
		// TODO url
		url := ""
		var ttMode reporter.TriggerTraceMode
		// TODO verify hmac signature
		if xto.TriggerTrace() {
			ttMode = reporter.ModeStrictTriggerTrace
		} else {
			ttMode = reporter.ModeTriggerTraceNotPresent
		}
		// TODO replace this section with a nicer oboe-like interface
		// TODO handle RecordOnly (metrics)
		traceDecision := s.decider.ShouldTraceRequestWithURL(parameters.Name, traced, url, ttMode)
		var decision sdktrace.SamplingDecision
		if traceDecision.Trace() {
			decision = sdktrace.RecordAndSample
		} else {
			decision = sdktrace.Drop
		}
		result = sdktrace.SamplingResult{
			Decision:   decision,
			Tracestate: psc.TraceState(),
		}
	}

	// TODO look at the additionalAttributesBuilder in the java implementation

	return result

}
