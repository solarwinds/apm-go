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
package sampler

import (
	"context"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/swotel"
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/testutils"
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/xtrace"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"testing"
	"time"

	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	traceId   = testutils.TraceID
	spanId    = testutils.SpanID
	spanIdHex = testutils.SpanIdHex
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

func TestScenarioSwKeys(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:        true,
		traceStateContainsOther: true,
		triggerTrace:            true,
		xtraceSignature:         false,
		xtraceSwKeys:            true,

		oboeDecision: true,
		ttMode:       reporter.ModeTriggerTraceNotPresent,
		decision:     sdktrace.RecordAndSample,
	}
	scen.test(t)
}

func TestScenarioSwKeysUnsampled(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:     true,
		traceStateContainsSw: true,
		traceStateSwSampled:  false,
		oboeDecision:         false,
		xtraceSwKeys:         true,

		ttMode:   reporter.ModeTriggerTraceNotPresent,
		decision: sdktrace.RecordOnly,
	}
	scen.test(t)
}

func TestScenarioCustomKeys(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:        true,
		traceStateContainsOther: true,
		triggerTrace:            true,
		xtraceSignature:         false,
		xtraceCustomKeys:        true,

		oboeDecision: true,
		ttMode:       reporter.ModeTriggerTraceNotPresent,
		decision:     sdktrace.RecordAndSample,
	}
	scen.test(t)
}

func TestScenarioCustomKeysUnsampled(t *testing.T) {
	scen := SamplingScenario{
		validTraceParent:     true,
		traceStateContainsSw: true,
		traceStateSwSampled:  false,
		oboeDecision:         false,
		xtraceCustomKeys:     false,

		ttMode:   reporter.ModeTriggerTraceNotPresent,
		decision: sdktrace.RecordOnly,
	}
	scen.test(t)
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

	triggerTrace     bool
	xtraceSignature  bool
	xtraceSwKeys     bool
	xtraceCustomKeys bool

	oboeDecision bool

	// expectations
	ttMode   reporter.TriggerTraceMode
	decision sdktrace.SamplingDecision
}

func requireAttrEqual(t *testing.T, attrs attribute.Set, key string, expected attribute.Value) {
	val, ok := attrs.Value(attribute.Key(key))
	require.True(t, ok)
	require.Equal(t, expected, val, "Expected %s; got %s", expected.Emit(), val.Emit())
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
		traceState, err = swotel.SetSw(traceState, fmt.Sprintf("2222222222222222-%s", flags))
		if err != nil {
			t.Fatal("Could not insert tracestate key")
		}
	}
	if s.traceStateContainsOther {
		traceState, err = traceState.Insert("other", "capture me!")
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
	var xopts []string
	if s.triggerTrace {
		xopts = append(xopts, "trigger-trace")
	}
	if s.xtraceSwKeys {
		xopts = append(xopts, "sw-keys=lo:se,check-id:123")
	}
	if s.xtraceCustomKeys {
		xopts = append(xopts, "custom-key1=value 1;custom-key2=value 2")
	}
	if len(xopts) > 0 {
		ctx = context.WithValue(ctx, xtrace.OptionsKey, strings.Join(xopts, ";"))
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

	attrs := attribute.NewSet(result.Attributes...)

	if result.Decision == sdktrace.RecordAndSample {
		bucketCap := "1000000"
		bucketRate := bucketCap
		sampleRate := 1000000
		sampleSource := reporter.SAMPLE_SOURCE_DEFAULT
		if s.triggerTrace && !s.traceStateSwSampled {
			sampleRate = -1
			sampleSource = reporter.SAMPLE_SOURCE_UNSET
		}
		if s.traceStateSwSampled {
			bucketCap, bucketRate, sampleRate, sampleSource = "-1", "-1", -1, -1
		}
		requireAttrEqual(t, attrs, "BucketCapacity", attribute.StringValue(bucketCap))
		requireAttrEqual(t, attrs, "BucketRate", attribute.StringValue(bucketRate))
		requireAttrEqual(t, attrs, "SampleRate", attribute.IntValue(sampleRate))
		requireAttrEqual(t, attrs, "SampleSource", attribute.IntValue(int(sampleSource)))
	}

	if s.traceStateContainsOther {
		if result.Decision == sdktrace.RecordAndSample {
			captured, ok := attrs.Value("sw.w3c.tracestate")
			require.True(t, ok)
			require.Equal(t, "other=capture me!", captured.AsString())
		} else {
			require.False(t, attrs.HasValue("s3.w3c.tracestate"))
		}
	}

	if s.xtraceSwKeys {
		swKeys, ok := attrs.Value("SWKeys")
		if result.Decision == sdktrace.RecordAndSample {
			require.True(t, ok)
			require.Equal(t, swKeys.AsString(), "lo:se,check-id:123")
		} else {
			require.False(t, ok)
			require.Equal(t, swKeys.AsString(), "")
		}
	}

	if s.xtraceCustomKeys {
		res := make(map[string]string)
		for _, a := range result.Attributes {
			if strings.HasPrefix(string(a.Key), "custom-") {
				res[string(a.Key)] = a.Value.AsString()
			}
		}
		if result.Decision == sdktrace.RecordAndSample {
			require.Equal(t, map[string]string{
				"custom-key1": "value 1",
				"custom-key2": "value 2",
			}, res)
		} else {
			require.Len(t, res, 0)
		}
	}

	if s.triggerTrace && result.Decision == sdktrace.RecordAndSample {
		v, ok := attrs.Value("TriggeredTrace")
		require.True(t, ok)
		require.Equal(t, "true", v.AsString())
	}
}

// hydrateTraceState

func TestHydrateTraceState(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,
	})
	ctx := context.WithValue(context.Background(), xtrace.OptionsKey, "trigger-trace")
	xto := xtrace.GetXTraceOptions(ctx)
	ts := hydrateTraceState(sc, xto, "ok")
	fullResp, err := swotel.GetInternalState(ts, swotel.XTraceOptResp)
	require.NoError(t, err)
	require.Equal(t, "trigger-trace=ok", fullResp)
}

func TestHydrateTraceStateBadTimestamp(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,
	})
	ctx := context.WithValue(context.Background(), xtrace.OptionsKey, "trigger-trace")
	ctx = context.WithValue(ctx, xtrace.SignatureKey, "not a valid signature")
	xto := xtrace.GetXTraceOptions(ctx)
	ts := hydrateTraceState(sc, xto, "")
	fullResp, err := swotel.GetInternalState(ts, swotel.XTraceOptResp)
	require.NoError(t, err)
	require.Equal(t, "auth=bad-timestamp", fullResp)
}

func TestHydrateTraceStateBadSignature(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,
	})
	opts := fmt.Sprintf("trigger-trace;ts=%d", time.Now().Unix())
	ctx := context.WithValue(context.Background(), xtrace.OptionsKey, opts)
	sig := "invalid signature"
	ctx = context.WithValue(ctx, xtrace.SignatureKey, sig)
	xto := xtrace.GetXTraceOptions(ctx)
	ts := hydrateTraceState(sc, xto, "")
	fullResp, err := swotel.GetInternalState(ts, swotel.XTraceOptResp)
	require.NoError(t, err)
	require.Equal(t, "auth=bad-signature", fullResp)
}

func TestHydrateTraceStateNoSignatureKey(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.NoSettingST))
	defer r.Close(0)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,
	})
	opts := fmt.Sprintf("trigger-trace;ts=%d", time.Now().Unix())
	ctx := context.WithValue(context.Background(), xtrace.OptionsKey, opts)
	sig := "0000"
	ctx = context.WithValue(ctx, xtrace.SignatureKey, sig)
	xto := xtrace.GetXTraceOptions(ctx)
	ts := hydrateTraceState(sc, xto, "ok")
	fullResp, err := swotel.GetInternalState(ts, swotel.XTraceOptResp)
	require.NoError(t, err)
	require.Equal(t, "auth=no-signature-key", fullResp)
}

func TestHydrateTraceStateValidSignature(t *testing.T) {
	// set test reporter so we can use the hmac token for signing the xto
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	defer r.Close(0)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,
	})
	opts := fmt.Sprintf("trigger-trace;ts=%d", time.Now().Unix())
	ctx := context.WithValue(context.Background(), xtrace.OptionsKey, opts)
	sig, err := reporter.HmacHashTT([]byte(opts))
	require.NoError(t, err)
	ctx = context.WithValue(ctx, xtrace.SignatureKey, sig)
	xto := xtrace.GetXTraceOptions(ctx)
	ts := hydrateTraceState(sc, xto, "ok")
	fullResp, err := swotel.GetInternalState(ts, swotel.XTraceOptResp)
	require.NoError(t, err)
	require.Equal(t, "auth=ok;trigger-trace=ok", fullResp)
}
