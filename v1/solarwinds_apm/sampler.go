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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Sampler struct{}

func (s *Sampler) Description() string {
	return "SolarWinds APM Sampler"
}

func newTraceState() trace.TraceState {
	ts := trace.TraceState{}
	ts.Insert(VendorID, "TODO")
	return ts
}

func (s *Sampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	parentContext := parameters.ParentContext
	traced := false

	psc := trace.SpanContextFromContext(parentContext)
	log.Info("Parent context", parentContext)
	psc.TraceState()
	if !psc.IsValid() {
		_ = newTraceState()
		psc.HasSpanID()
		psc.SpanID()
		psc.TraceID()
	}

	if psc.IsValid() {
		log.Info("valid: %#v", psc)
	}
	if psc.IsRemote() {
		log.Info("remote: %#v", psc)
	}
	if psc.IsValid() && psc.IsRemote() {
		log.Infof("remote: %#v", psc)
	}

	// TODO
	url := ""

	xtoValue := parentContext.Value("X-Trace-Options")
	xtosValue := parentContext.Value("X-Trace-Options-Signature")
	var ttMode reporter.TriggerTraceMode
	if xtoValue != nil && xtosValue != nil {
		log.Infof("Got xtrace: %s %s", xtoValue, xtosValue)
		ttMode = reporter.ParseTriggerTrace(xtoValue.(string), xtosValue.(string))
	} else {
		ttMode = reporter.ModeTriggerTraceNotPresent
	}
	traceDecision, _ := reporter.ShouldTraceRequestWithURL(parameters.Name, traced, url, ttMode)

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
