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

package oboe

import (
	"github.com/solarwinds/apm-go/internal/oboetestutils"
	"github.com/solarwinds/apm-go/internal/w3cfmt"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	sampledSwState   = w3cfmt.ParseSwTraceState("0123456789abcdef-01")
	unsampledSwState = w3cfmt.ParseSwTraceState("0123456789abcdef-00")
)

//func TestCreateInitMessage(t *testing.T) {
//	tid := trace.TraceID{0x01, 0x02, 0x03, 0x04}
//	r, err := resource.New(context.Background(), resource.WithAttributes(
//		attribute.String("foo", "bar"),
//		// service.name should be omitted
//		attribute.String("service.name", "my cool service"),
//	))
//	require.NoError(t, err)
//	a := time.Now()
//	evt := createInitMessage(tid, r)
//	b := time.Now()
//	require.NoError(t, err)
//	require.NotNil(t, evt)
//	e, ok := evt.(*event)
//	require.True(t, ok)
//	require.Equal(t, tid, e.taskID)
//	require.NotEqual(t, [8]byte{}, e.opID)
//	require.True(t, e.t.After(a))
//	require.True(t, e.t.Before(b))
//	require.Equal(t, []attribute.KeyValue{
//		attribute.String("foo", "bar"),
//		attribute.Bool("__Init", true),
//		attribute.String("APM.Version", utils.Version()),
//	}, e.kvs)
//	require.Equal(t, LabelUnset, e.label)
//	require.Equal(t, "", e.layer)
//	require.False(t, e.parent.IsValid())
//}

func TestOboeSampleRequestSettingsUnavailable(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		xTraceOptsRsp: "settings-not-available",
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestSettingsDisabled(t *testing.T) {
	ttMode := ModeRelaxedTriggerTrace
	o := NewOboe()
	oboetestutils.AddDisabled(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		xTraceOptsRsp: "tracing-disabled",
		bucketCap:     1,
		bucketRate:    1,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequest(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	oboetestutils.AddDefaultSetting(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          1000000,
		source:        SAMPLE_SOURCE_DEFAULT,
		enabled:       true,
		xTraceOptsRsp: "not-requested",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    true,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestContinuedUnsampledSwState(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	oboetestutils.AddDefaultSetting(o)
	dec := o.OboeSampleRequest(true, "url", ttMode, unsampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          1000000,
		source:        SAMPLE_SOURCE_DEFAULT,
		enabled:       true,
		xTraceOptsRsp: "not-requested",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestNoTTGivenButReporterIsTTOnly(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	oboetestutils.AddTriggerTraceOnly(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          0,
		source:        SAMPLE_SOURCE_DEFAULT,
		enabled:       false,
		xTraceOptsRsp: "not-requested",
		bucketCap:     0,
		bucketRate:    0,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestUnsampledSwState(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	oboetestutils.AddDefaultSetting(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, unsampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          1000000,
		source:        SAMPLE_SOURCE_DEFAULT,
		enabled:       true,
		xTraceOptsRsp: "not-requested",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    true,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestThrough(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	oboetestutils.AddSampleThrough(o)
	dec := o.OboeSampleRequest(true, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          1000000,
		source:        SAMPLE_SOURCE_DEFAULT,
		enabled:       true,
		xTraceOptsRsp: "not-requested",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    true,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestThroughUnsampled(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	oboetestutils.AddSampleThrough(o)
	dec := o.OboeSampleRequest(true, "url", ttMode, unsampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          1000000,
		source:        SAMPLE_SOURCE_DEFAULT,
		enabled:       true,
		xTraceOptsRsp: "not-requested",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

// TRIGGER TRACE

func TestOboeSampleRequestRelaxedTT(t *testing.T) {
	ttMode := ModeRelaxedTriggerTrace
	o := NewOboe()
	oboetestutils.AddDefaultSetting(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		enabled:       true,
		xTraceOptsRsp: "ok",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestStrictTT(t *testing.T) {
	ttMode := ModeStrictTriggerTrace
	o := NewOboe()
	oboetestutils.AddDefaultSetting(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		enabled:       true,
		xTraceOptsRsp: "ok",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestRelaxedTTDisabled(t *testing.T) {
	ttMode := ModeRelaxedTriggerTrace
	o := NewOboe()
	oboetestutils.AddNoTriggerTrace(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		enabled:       true,
		xTraceOptsRsp: "trigger-tracing-disabled",
		bucketCap:     0,
		bucketRate:    0,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestStrictTTDisabled(t *testing.T) {
	ttMode := ModeStrictTriggerTrace
	o := NewOboe()
	oboetestutils.AddNoTriggerTrace(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		enabled:       true,
		xTraceOptsRsp: "trigger-tracing-disabled",
		bucketCap:     0,
		bucketRate:    0,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestRelaxedTTLimited(t *testing.T) {
	ttMode := ModeRelaxedTriggerTrace
	o := NewOboe()
	oboetestutils.AddLimitedTriggerTrace(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	// We expect the first TT to go through
	expected := SampleDecision{
		trace:         true,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		enabled:       true,
		xTraceOptsRsp: "ok",
		bucketCap:     1,
		bucketRate:    1,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
	dec = o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected = SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		enabled:       true,
		xTraceOptsRsp: "rate-exceeded",
		bucketCap:     1,
		bucketRate:    1,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestInvalidTT(t *testing.T) {
	ttMode := ModeInvalidTriggerTrace
	o := NewOboe()
	oboetestutils.AddDefaultSetting(o)
	dec := o.OboeSampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SAMPLE_SOURCE_UNSET,
		enabled:       true,
		xTraceOptsRsp: "",
		bucketCap:     1000000,
		bucketRate:    1000000,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
}

func TestGetTokenBucketSetting(t *testing.T) {
	main := &tokenBucket{ratePerSec: 1, capacity: 2}
	relaxed := &tokenBucket{ratePerSec: 3, capacity: 4}
	strict := &tokenBucket{ratePerSec: 5, capacity: 6}
	setting := &oboeSettings{
		bucket:                    main,
		triggerTraceRelaxedBucket: relaxed,
		triggerTraceStrictBucket:  strict,
	}

	scenarios := []struct {
		mode   TriggerTraceMode
		bucket *tokenBucket
	}{
		{ModeRelaxedTriggerTrace, relaxed},
		{ModeStrictTriggerTrace, strict},
		{ModeTriggerTraceNotPresent, main},
		{ModeInvalidTriggerTrace, main},
		{99, nil},
	}
	for _, scen := range scenarios {
		capacity, rate := getTokenBucketSetting(setting, scen.mode)
		if scen.bucket == nil {
			require.Equal(t, float64(0), capacity)
			require.Equal(t, float64(0), rate)
		} else {
			require.Equal(t, scen.bucket.capacity, capacity)
			require.Equal(t, scen.bucket.ratePerSec, rate)
		}
	}
}
