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

//

// Note: this is a port of the Python apm tests, found here:
// https://github.com/solarwindscloud/solarwinds-apm-python/blob/main/tests/unit/test_xtraceoptions.py
package xtrace

import (
	"context"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetXTraceOptions(t *testing.T) {
	// We set the test reporter which will set the TT Token used for HMAC verification
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	defer r.Close(0)
	ctx := context.TODO()
	// Timestamp required in signature validation
	opts := fmt.Sprintf("sw-keys=check-id:check-1013,website-id;booking-demo;ts=%d", time.Now().Unix())
	ctx = context.WithValue(ctx, OptionsKey, opts)
	sig, err := reporter.HmacHashTT([]byte(opts))
	if err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, SignatureKey, sig)

	xto := GetXTraceOptions(ctx)
	assert.Equal(t, "check-id:check-1013,website-id", xto.SwKeys())
	assert.Equal(t, []string{"booking-demo"}, xto.IgnoredKeys())
	assert.Equal(t, sig, xto.Signature())
	assert.Equal(t, ValidSignature, xto.SignatureState())
}

func TestGetXTraceOptionsInvalidType(t *testing.T) {
	ctx := context.TODO()
	ctx = context.WithValue(ctx, OptionsKey, 123)
	ctx = context.WithValue(ctx, SignatureKey, 321)

	xto := GetXTraceOptions(ctx)
	assert.Equal(t, "", xto.SwKeys())
	assert.Equal(t, []string{}, xto.IgnoredKeys())
	assert.Equal(t, "", xto.Signature())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestNoKeyNoValue(t *testing.T) {
	xto := parseXTraceOptions("=", "")
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestOrphanValue(t *testing.T) {
	xto := parseXTraceOptions("=oops", "")
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestValidTT(t *testing.T) {
	xto := parseXTraceOptions("trigger-trace", "")
	assert.True(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestTTKeyIgnored(t *testing.T) {
	xto := parseXTraceOptions("trigger-trace=1", "")
	assert.False(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestSwKeysKVStrip(t *testing.T) {
	xto := parseXTraceOptions("sw-keys=   foo:key   ", "")
	assert.Equal(t, "foo:key", xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestSwKeysContainingSemicolonIgnoreAfter(t *testing.T) {
	xto := parseXTraceOptions("sw-keys=check-id:check-1013,website-id;booking-demo", "")
	assert.Equal(t, "check-id:check-1013,website-id", xto.SwKeys())
	assert.Equal(t, []string{"booking-demo"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestCustomKeysMatchStoredInOptionsHeaderAndCustomKVs(t *testing.T) {
	xto := parseXTraceOptions("custom-awesome-key=    foo ", "")
	assert.Equal(t, map[string]string{"custom-awesome-key": "foo"}, xto.CustomKVs())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestCustomKeysMatchButNoValueIgnored(t *testing.T) {
	xto := parseXTraceOptions("custom-no-value", "")
	assert.Equal(t, map[string]string{}, xto.CustomKVs())
	assert.Equal(t, []string{"custom-no-value"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestCustomKeysMatchEqualInValue(t *testing.T) {
	xto := parseXTraceOptions("custom-and=a-value=12345containing_equals=signs", "")
	assert.Equal(t, map[string]string{"custom-and": "a-value=12345containing_equals=signs"}, xto.CustomKVs())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestCustomKeysSpacesInKeyDisallowed(t *testing.T) {
	xto := parseXTraceOptions("custom- key=this_is_bad;custom-key 7=this_is_bad_too", "")
	assert.Equal(t, map[string]string{}, xto.CustomKVs())
	assert.Equal(t, []string{"custom- key", "custom-key 7"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestValidTs(t *testing.T) {
	xto := parseXTraceOptions("ts=12345", "")
	assert.Equal(t, int64(12345), xto.Timestamp())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestInvalidTs(t *testing.T) {
	xto := parseXTraceOptions("ts=invalid", "")
	assert.Equal(t, int64(0), xto.Timestamp())
	assert.Equal(t, []string{"ts"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestOtherKeyIgnored(t *testing.T) {
	xto := parseXTraceOptions("customer-key=foo", "")
	assert.Equal(t, []string{"customer-key"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestSig(t *testing.T) {
	xto := parseXTraceOptions("foo bar baz", "signature123")
	assert.Equal(t, "signature123", xto.Signature())
	assert.Equal(t, []string{"foo bar baz"}, xto.IgnoredKeys())
	assert.Equal(t, InvalidSignature, xto.SignatureState())
}

func TestSigWithoutOptions(t *testing.T) {
	xto := parseXTraceOptions("", "signature123")
	assert.Equal(t, "signature123", xto.Signature())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, InvalidSignature, xto.SignatureState())
}

func TestDocumentedExample1(t *testing.T) {
	xto := parseXTraceOptions("trigger-trace;sw-keys=check-id:check-1013,website-id:booking-demo", "")
	assert.True(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Equal(t, "check-id:check-1013,website-id:booking-demo", xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestDocumentedExample2(t *testing.T) {
	xto := parseXTraceOptions("trigger-trace;custom-key1=value1", "")
	assert.True(t, xto.TriggerTrace())
	assert.Equal(t, map[string]string{"custom-key1": "value1"}, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestDocumentedExample3(t *testing.T) {
	xto := parseXTraceOptions(
		"trigger-trace;sw-keys=check-id:check-1013,website-id:booking-demo;ts=1564432370",
		"5c7c733c727e5038d2cd537630206d072bbfc07c",
	)
	assert.True(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Equal(t, "check-id:check-1013,website-id:booking-demo", xto.SwKeys())
	assert.Equal(t, int64(1564432370), xto.Timestamp())
	assert.Empty(t, xto.IgnoredKeys())
	assert.Equal(t, InvalidSignature, xto.SignatureState())
}

func TestStripAllOptions(t *testing.T) {
	xto := parseXTraceOptions(
		" trigger-trace ;  custom-something=value; custom-OtherThing = other val ;  sw-keys = 029734wr70:9wqj21,0d9j1   ; ts = 12345 ; foo = bar ",
		"",
	)
	assert.Empty(t, xto.Signature())
	assert.Equal(t, map[string]string{
		"custom-something":  "value",
		"custom-OtherThing": "other val",
	}, xto.CustomKVs())
	assert.Equal(t, "029734wr70:9wqj21,0d9j1", xto.SwKeys())
	assert.True(t, xto.TriggerTrace())
	assert.Equal(t, int64(12345), xto.Timestamp())
	assert.Equal(t, []string{"foo"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestAllOptionsHandleSequentialSemicolons(t *testing.T) {
	xto := parseXTraceOptions(
		";foo=bar;;;custom-something=value_thing;;sw-keys=02973r70:1b2a3;;;;custom-key=val;ts=12345;;;;;;;trigger-trace;;;",
		"",
	)
	assert.Empty(t, xto.Signature())
	assert.Equal(t, map[string]string{
		"custom-something": "value_thing",
		"custom-key":       "val",
	}, xto.CustomKVs())
	assert.Equal(t, "02973r70:1b2a3", xto.SwKeys())
	assert.True(t, xto.TriggerTrace())
	assert.Equal(t, int64(12345), xto.Timestamp())
	assert.Equal(t, []string{"foo"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestAllOptionsHandleSingleQuotes(t *testing.T) {
	xto := parseXTraceOptions(
		"trigger-trace;custom-foo='bar;bar';custom-bar=foo",
		"",
	)
	assert.Empty(t, xto.Signature())
	assert.Equal(t, map[string]string{
		"custom-foo": "'bar",
		"custom-bar": "foo",
	}, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.True(t, xto.TriggerTrace())
	assert.Equal(t, int64(0), xto.Timestamp())
	assert.Equal(t, []string{"bar'"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}

func TestAllOptionsHandleMissingValuesAndSemicolons(t *testing.T) {
	xto := parseXTraceOptions(
		";trigger-trace;custom-something=value_thing;sw-keys=02973r70:9wqj21,0d9j1;1;2;3;4;5;=custom-key=val?;=",
		"",
	)
	assert.Empty(t, xto.Signature())
	assert.Equal(t, map[string]string{
		"custom-something": "value_thing",
	}, xto.CustomKVs())
	assert.Equal(t, "02973r70:9wqj21,0d9j1", xto.SwKeys())
	assert.True(t, xto.TriggerTrace())
	assert.Equal(t, int64(0), xto.Timestamp())
	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, xto.IgnoredKeys())
	assert.Equal(t, NoSignature, xto.SignatureState())
}
