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
package config

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidServiceKey(t *testing.T) {
	valid1 := "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:Go"
	valid2 := "ae38-315f611658_5d64d82eW-c2455aa3NPec61e02fee25d2D86f74ace9e4fea189217:Go"

	invalid1 := ""
	invalid2 := "abc:Go"
	invalid3 := `
ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:
Go0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
`
	invalid4 := "1234567890abcdef"
	invalid5 := "1234567890abcdef:"
	invalid6 := ":Go"
	invalid7 := "abc:123:Go"

	keyPairs := map[string]bool{
		valid1:   true,
		valid2:   true,
		invalid1: false,
		invalid2: false,
		invalid3: false,
		invalid4: false,
		invalid5: false,
		invalid6: false,
		invalid7: false,
	}

	for key, valid := range keyPairs {
		assert.Equal(t, valid, IsValidServiceKey(key))
	}
}

func TestMaskServiceKey(t *testing.T) {
	keyPairs := map[string]string{
		"1234567890abcdef:Go": "1234********cdef:Go",
		"abc:Go":              "abc:Go",
		"abcd1234:Go":         "abcd1234:Go",
	}

	for key, masked := range keyPairs {
		assert.Equal(t, masked, MaskServiceKey(key))
	}
}

func TestIsValidTracingMode(t *testing.T) {
	assert.Equal(t, true, IsValidTracingMode("enabled"))
	assert.Equal(t, true, IsValidTracingMode("disabled"))
	assert.Equal(t, false, IsValidTracingMode("abc"))
	assert.Equal(t, false, IsValidTracingMode(""))
	assert.Equal(t, false, IsValidTracingMode("ENABLED"))
	assert.Equal(t, false, IsValidTracingMode("ALWAYS"))
	assert.Equal(t, false, IsValidTracingMode("NEVER"))
}

func TestConverters(t *testing.T) {
	assert.Equal(t, DisabledTracingMode, NormalizeTracingMode("disabled"))
	assert.Equal(t, DisabledTracingMode, NormalizeTracingMode("never"))
	assert.Equal(t, EnabledTracingMode, NormalizeTracingMode("always"))
	assert.Equal(t, EnabledTracingMode, NormalizeTracingMode("ALWAYS"))
	assert.Equal(t, DisabledTracingMode, NormalizeTracingMode("NEVER"))
}

func withDemoKey(sn string) string {
	return "demo_service_key:" + sn
}

func TestToServiceKey(t *testing.T) {
	cases := []struct{ before, after string }{
		{withDemoKey("hello"), withDemoKey("hello")},
		{withDemoKey("he llo"), withDemoKey("he-llo")},
		{withDemoKey("he	llo"), withDemoKey("he-llo")},
		{withDemoKey(" he llo "), withDemoKey("-he-llo-")},
		{withDemoKey("HE llO "), withDemoKey("he-llo-")},
		{withDemoKey("hE~ l * "), withDemoKey("he-l--")},
		{withDemoKey("*^&$"), withDemoKey("")},
		{withDemoKey("he  llo"), withDemoKey("he--llo")},
		{withDemoKey("a:b"), withDemoKey("a:b")},
		{withDemoKey(":"), withDemoKey(":")},
		{withDemoKey(":::"), withDemoKey(":::")},
		{"badServiceKey", "badServiceKey"},
		{"badServiceKey:", "badServiceKey:"},
		{":badServiceKey", ":badservicekey"},
		{"", ""},
	}
	for idx, tc := range cases {
		assert.Equal(t, tc.after, ToServiceKey(tc.before), fmt.Sprintf("Case #%d", idx))
	}
}

func TestIsValidHost(t *testing.T) {
	require.True(t, IsValidHost("localhost"))
	require.True(t, IsValidHost("localhost:321"))
	require.True(t, IsValidHost("[2001:db8::ff00:42:8329]"))
	require.True(t, IsValidHost("[2001:db8::ff00:42:8329]:1234"))
	require.False(t, IsValidHost(""))
	require.False(t, IsValidHost("localhost:321:321"))
	require.False(t, IsValidHost("2001:db8::ff00:42:8329"))
	require.False(t, IsValidHost("2001:db8::ff00:42:8329:1234"))
}
