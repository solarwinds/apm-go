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
	"testing"
	"time"

	"github.com/solarwinds/apm-go/internal/w3cfmt"
	"github.com/stretchr/testify/require"
)

var (
	sampledSwState   = w3cfmt.ParseSwTraceState("0123456789abcdef-01")
	unsampledSwState = w3cfmt.ParseSwTraceState("0123456789abcdef-00")
)

func GetDefaultSettingForTest() SettingsUpdateArgs {
	return SettingsUpdateArgs{
		Flags:                        "SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE",
		Value:                        1000000,
		Ttl:                          120 * time.Second,
		TriggerToken:                 []byte("token"),
		BucketCapacity:               1000000,
		BucketRate:                   1000000,
		MetricsFlushInterval:         -1,
		TriggerRelaxedBucketCapacity: 1000000,
		TriggerRelaxedBucketRate:     1000000,
		TriggerStrictBucketCapacity:  1000000,
		TriggerStrictBucketRate:      1000000,
	}
}

func (settings SettingsUpdateArgs) WithSampleThrough() SettingsUpdateArgs {
	settings.Flags = "SAMPLE_START,SAMPLE_THROUGH,TRIGGER_TRACE"
	return settings
}

func (settings SettingsUpdateArgs) WithDisabled() SettingsUpdateArgs {
	settings.Flags = ""
	settings.BucketCapacity = 0
	settings.BucketRate = 0
	settings.TriggerRelaxedBucketCapacity = 1
	settings.TriggerRelaxedBucketRate = 1
	settings.TriggerStrictBucketCapacity = 1
	settings.TriggerStrictBucketRate = 1
	return settings
}

func (settings SettingsUpdateArgs) WithTriggerTraceOnly() SettingsUpdateArgs {
	settings.Flags = "TRIGGER_TRACE"
	settings.Value = 0
	settings.BucketCapacity = 0
	settings.BucketRate = 0
	return settings
}

func (settings SettingsUpdateArgs) WithNoTriggerTrace() SettingsUpdateArgs {
	settings.Flags = "SAMPLE_START,SAMPLE_THROUGH_ALWAYS"
	settings.TriggerRelaxedBucketCapacity = 0
	settings.TriggerRelaxedBucketRate = 0
	settings.TriggerStrictBucketCapacity = 0
	settings.TriggerStrictBucketRate = 0
	return settings
}

func (settings SettingsUpdateArgs) WithLimitedTriggerTrace() SettingsUpdateArgs {
	settings.Flags = "SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE"
	settings.TriggerRelaxedBucketCapacity = 1
	settings.TriggerRelaxedBucketRate = 1
	settings.TriggerStrictBucketCapacity = 1
	settings.TriggerStrictBucketRate = 1
	return settings
}

func TestOboeSampleRequestSettingsUnavailable(t *testing.T) {
	ttMode := ModeTriggerTraceNotPresent
	o := NewOboe()
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		xTraceOptsRsp: "settings-not-available",
	}
	require.Equal(t, expected, dec)
}

func TestOboeSampleRequestSettingsDisabled(t *testing.T) {
	ttMode := ModeRelaxedTriggerTrace
	o := NewOboe()
	o.UpdateSetting(GetDefaultSettingForTest().WithDisabled())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SampleSourceUnset,
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
	o.UpdateSetting(GetDefaultSettingForTest())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          1000000,
		source:        SampleSourceDefault,
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
	o.UpdateSetting(GetDefaultSettingForTest())
	dec := o.SampleRequest(true, "url", ttMode, unsampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          1000000,
		source:        SampleSourceDefault,
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
	o.UpdateSetting(GetDefaultSettingForTest().WithTriggerTraceOnly())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          0,
		source:        SampleSourceDefault,
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
	o.UpdateSetting(GetDefaultSettingForTest())
	dec := o.SampleRequest(false, "url", ttMode, unsampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          1000000,
		source:        SampleSourceDefault,
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
	o.UpdateSetting(GetDefaultSettingForTest().WithSampleThrough())
	dec := o.SampleRequest(true, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          1000000,
		source:        SampleSourceDefault,
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
	o.UpdateSetting(GetDefaultSettingForTest().WithSampleThrough())
	dec := o.SampleRequest(true, "url", ttMode, unsampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          1000000,
		source:        SampleSourceDefault,
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
	o.UpdateSetting(GetDefaultSettingForTest())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          -1,
		source:        SampleSourceUnset,
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
	o.UpdateSetting(GetDefaultSettingForTest())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         true,
		rate:          -1,
		source:        SampleSourceUnset,
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
	o.UpdateSetting(GetDefaultSettingForTest().WithNoTriggerTrace())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SampleSourceUnset,
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
	o.UpdateSetting(GetDefaultSettingForTest().WithNoTriggerTrace())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SampleSourceUnset,
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
	o.UpdateSetting(GetDefaultSettingForTest().WithLimitedTriggerTrace())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	// We expect the first TT to go through
	expected := SampleDecision{
		trace:         true,
		rate:          -1,
		source:        SampleSourceUnset,
		enabled:       true,
		xTraceOptsRsp: "ok",
		bucketCap:     1,
		bucketRate:    1,
		diceRolled:    false,
	}
	require.Equal(t, expected, dec)
	dec = o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected = SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SampleSourceUnset,
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
	o.UpdateSetting(GetDefaultSettingForTest())
	dec := o.SampleRequest(false, "url", ttMode, sampledSwState)
	expected := SampleDecision{
		trace:         false,
		rate:          -1,
		source:        SampleSourceUnset,
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
	setting := &settings{
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
		capacity, rate := setting.getTokenBucketSetting(scen.mode)
		if scen.bucket == nil {
			require.Equal(t, float64(0), capacity)
			require.Equal(t, float64(0), rate)
		} else {
			require.Equal(t, scen.bucket.capacity, capacity)
			require.Equal(t, scen.bucket.ratePerSec, rate)
		}
	}
}
