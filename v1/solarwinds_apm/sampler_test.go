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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/w3cfmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	"go.opentelemetry.io/otel/trace"
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestNewSampler(t *testing.T) {
	s := NewSampler().(sampler)
	assert.Equal(t, s.decider, defaultDecider)
}

func TestDescription(t *testing.T) {
	s := NewSampler()
	assert.Equal(t, "SolarWinds APM Sampler", s.Description())
}

type expectedArgs struct {
	layer   string
	traced  bool
	url     string
	ttMode  reporter.TriggerTraceMode
	swState w3cfmt.SwTraceState
	t       *testing.T
}

type mockDecider struct {
	retval reporter.SampleDecision
	expectedArgs
	called bool
}

func (m *mockDecider) ShouldTraceRequestWithURL(
	layer string, traced bool, url string, ttMode reporter.TriggerTraceMode, swState *w3cfmt.SwTraceState) reporter.SampleDecision {
	assert.Equal(m.t, m.expectedArgs.layer, layer, "layer")
	assert.Equal(m.t, m.expectedArgs.traced, traced, "traced")
	assert.Equal(m.t, m.expectedArgs.url, url, "url")
	assert.Equal(m.t, m.expectedArgs.ttMode, ttMode, "ttMode")

	assert.Equal(m.t, m.expectedArgs.swState.Flags(), swState.Flags(), "swstate flags")
	assert.Equal(m.t, m.expectedArgs.swState.IsValid(), swState.IsValid(), "swstate is valid")
	assert.Equal(m.t, m.expectedArgs.swState.SpanId(), swState.SpanId(), "swstate span id")

	m.called = true
	return m.retval
}

func (m *mockDecider) Called() bool {
	return m.called
}

var _ decider = &mockDecider{}

// Input Headers - None
// Start a new trace decision
// Sets X-Trace response header
func TestScenario1(t *testing.T) {
	scen := SamplingScenario{
		oboeConsulted: true,
		decision:      sdktrace.RecordAndSample,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-111-01
// start a new trace decision
func TestScenario2(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent: true,
		oboeConsulted:    true,
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
		oboeConsulted:        true,
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
		oboeConsulted:        true,

		decision: sdktrace.RecordAndSample,
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
		oboeConsulted:           true,
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

		oboeConsulted: true,
		ttMode:        reporter.ModeStrictTriggerTrace,
		decision:      sdktrace.RecordAndSample,
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

		oboeConsulted: true,
		ttMode:        reporter.ModeStrictTriggerTrace,
		decision:      sdktrace.RecordAndSample,
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

		oboeConsulted: true,
		ttMode:        reporter.ModeStrictTriggerTrace,
		decision:      sdktrace.RecordAndSample,
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

		oboeConsulted: false,
	}
	scen.test(t)

	scen = SamplingScenario{
		validTraceParent: false,
		local:            true,

		oboeConsulted: true,
		decision:      sdktrace.RecordAndSample,
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

	// expectations
	oboeConsulted bool // Should call oboe's sampling logic
	obeyTT        bool // TODO verify this somehow (NH-5731)
	ttMode        reporter.TriggerTraceMode
	decision      sdktrace.SamplingDecision
}

func (s SamplingScenario) test(t *testing.T) {
	var err error
	decision := reporter.NewSampleDecision(true)
	swState := w3cfmt.SwTraceState{}
	// swState is the expected version we call oboe with
	if s.validTraceParent && s.traceStateContainsSw {
		flags := "00"
		if s.traceStateSwSampled {
			flags = "01"
		}
		swState = w3cfmt.ParseSwTraceState(fmt.Sprintf("2222222222222222-%s", flags))
	}
	mock := &mockDecider{
		retval: decision,
		expectedArgs: expectedArgs{
			ttMode:  s.ttMode,
			t:       t,
			traced:  s.validTraceParent && s.traceStateContainsSw,
			swState: swState,
		},
	}

	smplr := sampler{decider: mock}
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
	if s.xtraceSignature {
		// TODO do hmac (NH-5731)
		//ctx = context.WithValue(ctx, xtrace.SignatureKey, s.xtraceSig)
	}
	params := sdktrace.SamplingParameters{
		ParentContext: ctx,
		TraceID:       traceId,
	}
	result := smplr.ShouldSample(params)
	assert.Equal(t, s.decision, result.Decision)
	assert.Equal(t, s.oboeConsulted, mock.Called())
}
