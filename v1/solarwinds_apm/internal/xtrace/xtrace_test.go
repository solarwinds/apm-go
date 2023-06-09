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
package xtrace_test

import (
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	"github.com/stretchr/testify/assert"
)

func TestNoKeyNoValue(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("=", "")
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestOrphanValue(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("=oops", "")
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestValidTT(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("trigger-trace", "")
	assert.True(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestTTKeyIgnored(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("trigger-trace=1", "")
	assert.False(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestSwKeysKVStrip(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("sw-keys=   foo:key   ", "")
	assert.Equal(t, "foo:key", xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestSwKeysContainingSemicolonIgnoreAfter(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("sw-keys=check-id:check-1013,website-id;booking-demo", "")
	assert.Equal(t, "check-id:check-1013,website-id", xto.SwKeys())
	assert.Equal(t, []string{"booking-demo"}, xto.IgnoredKeys())
}

func TestCustomKeysMatchStoredInOptionsHeaderAndCustomKVs(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("custom-awesome-key=    foo ", "")
	assert.Equal(t, map[string]string{"custom-awesome-key": "foo"}, xto.CustomKVs())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestCustomKeysMatchButNoValueIgnored(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("custom-no-value", "")
	assert.Equal(t, map[string]string{}, xto.CustomKVs())
	assert.Equal(t, []string{"custom-no-value"}, xto.IgnoredKeys())
}

func TestCustomKeysMatchEqualInValue(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("custom-and=a-value=12345containing_equals=signs", "")
	assert.Equal(t, map[string]string{"custom-and": "a-value=12345containing_equals=signs"}, xto.CustomKVs())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestCustomKeysSpacesInKeyDisallowed(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("custom- key=this_is_bad;custom-key 7=this_is_bad_too", "")
	assert.Equal(t, map[string]string{}, xto.CustomKVs())
	assert.Equal(t, []string{"custom- key", "custom-key 7"}, xto.IgnoredKeys())
}

func TestValidTs(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("ts=12345", "")
	assert.Equal(t, int64(12345), xto.Timestamp())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestInvalidTs(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("ts=invalid", "")
	assert.Equal(t, int64(0), xto.Timestamp())
	assert.Equal(t, []string{"ts"}, xto.IgnoredKeys())
}

func TestOtherKeyIgnored(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("customer-key=foo", "")
	assert.Equal(t, []string{"customer-key"}, xto.IgnoredKeys())
}

func TestSig(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("foo bar baz", "signature123")
	assert.Equal(t, "signature123", xto.Signature())
	assert.Equal(t, []string{"foo bar baz"}, xto.IgnoredKeys())
}

func TestSigWithoutOptions(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("", "signature123")
	assert.Equal(t, "signature123", xto.Signature())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestDocumentedExample1(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("trigger-trace;sw-keys=check-id:check-1013,website-id:booking-demo", "")
	assert.True(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Equal(t, "check-id:check-1013,website-id:booking-demo", xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestDocumentedExample2(t *testing.T) {
	xto := xtrace.ParseXTraceOptions("trigger-trace;custom-key1=value1", "")
	assert.True(t, xto.TriggerTrace())
	assert.Equal(t, map[string]string{"custom-key1": "value1"}, xto.CustomKVs())
	assert.Empty(t, xto.SwKeys())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestDocumentedExample3(t *testing.T) {
	xto := xtrace.ParseXTraceOptions(
		"trigger-trace;sw-keys=check-id:check-1013,website-id:booking-demo;ts=1564432370",
		"5c7c733c727e5038d2cd537630206d072bbfc07c",
	)
	assert.True(t, xto.TriggerTrace())
	assert.Empty(t, xto.CustomKVs())
	assert.Equal(t, "check-id:check-1013,website-id:booking-demo", xto.SwKeys())
	assert.Equal(t, int64(1564432370), xto.Timestamp())
	assert.Empty(t, xto.IgnoredKeys())
}

func TestStripAllOptions(t *testing.T) {
	xto := xtrace.ParseXTraceOptions(
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
}

func TestAllOptionsHandleSequentialSemicolons(t *testing.T) {
	xto := xtrace.ParseXTraceOptions(
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
}

func TestAllOptionsHandleSingleQuotes(t *testing.T) {
	xto := xtrace.ParseXTraceOptions(
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
}

func TestAllOptionsHandleMissingValuesAndSemicolons(t *testing.T) {
	xto := xtrace.ParseXTraceOptions(
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
}
