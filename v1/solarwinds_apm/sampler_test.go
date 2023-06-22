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
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/constants"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	"go.opentelemetry.io/otel/trace"
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestDescription(t *testing.T) {
	s := NewSampler()
	assert.Equal(t, "SolarWinds APM Sampler", s.Description())
}

// Input Headers - None
// Start a new trace decision
// Sets X-Trace response header
func TestScenario1(t *testing.T) {
	scen := SamplingScenario{
		oboeDecision: true,
		decision:     sdktrace.RecordAndSample,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-111-01
// start a new trace decision
func TestScenario2(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent: true,
		oboeDecision:     true,
		decision:         sdktrace.RecordAndSample,
	}
	scen.test(t)
}

// no traceparent
// non-empty tracestate
// start a new trace decision
func TestScenario3(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:     false,
		traceStateContainsSw: true,
		oboeDecision:         true,
		decision:             sdktrace.RecordAndSample,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-222-01
// valid tracestate with our vendor entry
// continue trace decision from sw value in tracestate
func TestScenario4(t *testing.T) {
	// sampled
	scen := SamplingScenario{
		validTraceParent:     true,
		traceStateContainsSw: true,
		traceStateSwSampled:  true,
		oboeDecision:         true,

		decision: sdktrace.RecordAndSample,
	}
	scen.test(t)

	// not sampled
	scen = SamplingScenario{
		validTraceParent:     true,
		traceStateContainsSw: true,
		traceStateSwSampled:  false,
		oboeDecision:         false,

		decision: sdktrace.RecordOnly,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-111-01
// valid tracestate without our vendor entry
// start a new trace decision
func TestScenario5(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:        true,
		traceStateContainsOther: true,
		oboeDecision:            true,
		decision:                sdktrace.RecordAndSample,
	}
	scen.test(t)
}

// no traceparent
// valid unsigned trigger trace x-trace-options: trigger-trace
// obey trigger trace rules
func TestScenario6(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent: false,
		triggerTrace:     true,
		xtraceSignature:  false,

		oboeDecision: true,
		ttMode:       reporter.ModeStrictTriggerTrace,
		decision:     sdktrace.RecordAndSample,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-111-01
// valid tracestate without our vendor entry
// valid unsigned trigger trace x-trace-options: trigger-trace
// obey trigger trace rules
func TestScenario7(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:        true,
		traceStateContainsOther: true,
		triggerTrace:            true,
		xtraceSignature:         false,

		oboeDecision: true,
		ttMode:       reporter.ModeStrictTriggerTrace,
		decision:     sdktrace.RecordAndSample,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-111-01
// valid tracestate with our vendor entry
// valid unsigned trigger trace x-trace-options: trigger-trace
// continue trace decision from sw value in tracestate
func TestScenario8(t *testing.T) {
	// sampled
	scen := SamplingScenario{
		validTraceParent:     true,
		traceStateContainsSw: true,
		traceStateSwSampled:  true,
		triggerTrace:         true,
		xtraceSignature:      false,

		oboeDecision: true,
		ttMode:       reporter.ModeStrictTriggerTrace,
		decision:     sdktrace.RecordAndSample,
	}
	scen.test(t)

	// No need to test unsampled here since oboe handles that logic
}

func TestLocalTraces(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:     true,
		traceStateContainsSw: true,
		traceStateSwSampled:  true,
		triggerTrace:         true,
		local:                true,
	}
	scen.test(t)

	scen = SamplingScenario{
		validTraceParent: false,
		local:            true,

		oboeDecision: true,
		decision:     sdktrace.RecordAndSample,
	}
	scen.test(t)
}

type SamplingScenario struct {
	// inputs
	validTraceParent        bool
	traceStateContainsSw    bool
	traceStateSwSampled     bool
	traceStateContainsOther bool
	local                   bool

	triggerTrace    bool
	xtraceSignature bool

	oboeDecision bool

	// expectations
	//obeyTT   bool // TODO verify this somehow (NH-5731)
	ttMode   reporter.TriggerTraceMode
	decision sdktrace.SamplingDecision
}

func (s SamplingScenario) test(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	defer r.Close(0)
	var err error

	smplr := NewSampler()
	traceState := trace.TraceState{}
	if s.traceStateContainsSw {
		flags := "00"
		if s.traceStateSwSampled {
			flags = "01"
		}
		traceState, err = traceState.Insert(constants.SWTraceStateKey, fmt.Sprintf("2222222222222222-%s", flags))
		if err != nil {
			t.Fatal("Could not insert tracestate key")
		}
	}
	if s.traceStateContainsOther {
		traceState, err = traceState.Insert("other", "this value should be ignored")
		if err != nil {
			t.Fatal("Could not insert tracestate key")
		}
	}

	spanCtxConfig := trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x00},
		SpanID:     trace.SpanID{0x00},
		TraceFlags: trace.TraceFlags(0x00),
		TraceState: traceState,
		Remote:     !s.local,
	}
	if s.validTraceParent {
		spanCtxConfig.TraceID = trace.TraceID{0x01}
		spanCtxConfig.SpanID = trace.SpanID{0x02}
	}
	spanCtx := trace.NewSpanContext(spanCtxConfig)
	assert.Equal(t, s.validTraceParent, spanCtx.IsValid())
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	if s.triggerTrace {
		ctx = context.WithValue(ctx, xtrace.OptionsKey, "trigger-trace")
	}
	//if s.xtraceSignature {
	//	// TODO do hmac (NH-5731)
	//	//ctx = context.WithValue(ctx, xtrace.SignatureKey, s.xtraceSig)
	//}
	params := sdktrace.SamplingParameters{
		ParentContext: ctx,
		TraceID:       traceId,
	}
	result := smplr.ShouldSample(params)
	assert.Equal(t, s.decision, result.Decision)
}