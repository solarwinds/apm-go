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
	ShouldTraceRequestWithURL(layer string, traced bool, url string, ttMode reporter.TriggerTraceMode) (bool, string)
}

type reporterDecider struct{}

func (r reporterDecider) ShouldTraceRequestWithURL(layer string, traced bool, url string, ttMode reporter.TriggerTraceMode) (bool, string) {
	return reporter.ShouldTraceRequestWithURL(layer, traced, url, ttMode)
}

var defaultDecider decider = reporterDecider{}

var _ sdktrace.Sampler = sampler{}

func (s sampler) Description() string {
	return "SolarWinds APM Sampler"
}

func (s sampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	parentContext := parameters.ParentContext
	traced := false

	psc := trace.SpanContextFromContext(parentContext)

	// TODO
	url := ""

	// TODO: re-examine whether this x-trace logic is valid. This was imported
	// from some test code before my time -jared
	ttMode := reporter.ModeTriggerTraceNotPresent
	if parentContext != nil {
		xtoValue := parentContext.Value("X-Trace-Options")
		xtosValue := parentContext.Value("X-Trace-Options-Signature")
		if xtoValue != nil && xtosValue != nil {
			ttMode = reporter.ParseTriggerTrace(xtoValue.(string), xtosValue.(string))
		}
	}
	traceDecision, _ := s.decider.ShouldTraceRequestWithURL(parameters.Name, traced, url, ttMode)

	var decision sdktrace.SamplingDecision
	if traceDecision {
		decision = sdktrace.RecordAndSample
	} else {
		decision = sdktrace.Drop
	}

	res := sdktrace.SamplingResult{
		Decision:   decision,
		Tracestate: psc.TraceState(),
	}
	return res

}
