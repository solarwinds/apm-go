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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	"go.opentelemetry.io/otel/trace"
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type staticDecider struct {
	retval reporter.SampleDecision
}

func (d staticDecider) ShouldTraceRequestWithURL(
	layer string, traced bool, url string, ttMode reporter.TriggerTraceMode) reporter.SampleDecision {
	return d.retval
}

// Note: I don't love these, but they work for this simple case.
var trueDecider decider = staticDecider{reporter.NewSampleDecision(true)}
var falseDecider decider = staticDecider{reporter.NewSampleDecision(false)}

func TestShouldSample(t *testing.T) {
	s := sampler{decider: trueDecider}
	params := sdktrace.SamplingParameters{
		ParentContext: context.TODO(),
	}
	result := s.ShouldSample(params)
	assert.Equal(t, sdktrace.RecordAndSample, result.Decision)
	// todo: figure out how to test resulting tracestate
}

func TestShouldSampleFalse(t *testing.T) {
	s := sampler{decider: falseDecider}
	params := sdktrace.SamplingParameters{
		ParentContext: context.TODO(),
	}
	result := s.ShouldSample(params)
	assert.Equal(t, sdktrace.Drop, result.Decision)
}

func TestNewSampler(t *testing.T) {
	s := NewSampler().(sampler)
	assert.Equal(t, s.decider, defaultDecider)
}

func TestDescription(t *testing.T) {
	s := NewSampler()
	assert.Equal(t, "SolarWinds APM Sampler", s.Description())
}

type expectedArgs struct {
	layer  string
	traced bool
	url    string
	ttMode reporter.TriggerTraceMode
	t      *testing.T
}

type mockDecider struct {
	retval reporter.SampleDecision
	expectedArgs
	called bool
}

func (m *mockDecider) ShouldTraceRequestWithURL(
	layer string, traced bool, url string, ttMode reporter.TriggerTraceMode) reporter.SampleDecision {
	assert.Equal(m.t, m.expectedArgs.layer, layer)
	assert.Equal(m.t, m.expectedArgs.traced, traced)
	assert.Equal(m.t, m.expectedArgs.url, url)
	assert.Equal(m.t, m.expectedArgs.ttMode, ttMode)
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
		newTraceDecision: true,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-111-01
// start a new trace decision
func TestScenario2(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent: true,
		newTraceDecision: true,
	}
	scen.test(t)
}

// no traceparent
// non-empty tracestate
// start a new trace decision
func TestScenario3(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:      false,
		traceStateContainerSw: true,
		newTraceDecision:      true,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-222-01
// valid tracestate with our vendor entry
// continue trace decision from sw value in tracestate
func TestScenario4(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:      true,
		traceStateContainerSw: true,
		newTraceDecision:      false,
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
		newTraceDecision:        true,
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

		newTraceDecision: true,
		ttMode:           reporter.ModeStrictTriggerTrace,
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

		newTraceDecision: true,
		ttMode:           reporter.ModeStrictTriggerTrace,
	}
	scen.test(t)
}

// valid traceparent 00-aaaaaa-111-01
// valid tracestate with our vendor entry
// valid unsigned trigger trace x-trace-options: trigger-trace
// continue trace decision from sw value in tracestate
func TestScenario8(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:      true,
		traceStateContainerSw: true,
		triggerTrace:          true,
		xtraceSignature:       false,

		newTraceDecision: false,
		ttMode:           reporter.ModeStrictTriggerTrace,
	}
	scen.test(t)
}

type SamplingScenario struct {
	// inputs
	validTraceParent        bool
	traceStateContainerSw   bool
	traceStateContainsOther bool

	triggerTrace    bool
	xtraceSignature bool

	// expectations
	newTraceDecision bool // Should call oboe's sampling logic
	obeyTT           bool // TODO verify this somehow
	ttMode           reporter.TriggerTraceMode
}

func (s SamplingScenario) test(t *testing.T) {
	var err error
	decision := reporter.NewSampleDecision(true)
	mock := &mockDecider{
		retval: decision,
		expectedArgs: expectedArgs{
			ttMode: s.ttMode,
			t:      t,
		},
	}

	smplr := sampler{decider: mock}
	traceState := trace.TraceState{}
	if s.traceStateContainerSw {
		// TODO test when tracestate flags are `unsampled`
		traceState, err = traceState.Insert(constants.SWTraceStateKey, "2222222222222222-01")
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
		Remote:     false,
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
		// TODO do hmac
		//ctx = context.WithValue(ctx, xtrace.SignatureKey, s.xtraceSig)
	}
	params := sdktrace.SamplingParameters{
		ParentContext: ctx,
		TraceID:       traceId,
	}
	result := smplr.ShouldSample(params)
	assert.Equal(t, sdktrace.RecordAndSample, result.Decision)
	assert.Equal(t, s.newTraceDecision, mock.Called())
}
