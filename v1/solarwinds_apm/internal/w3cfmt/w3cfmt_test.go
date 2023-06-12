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
package w3cfmt

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

const spanIdHex = "0123456789abcdef"

var spanId, err = trace.SpanIDFromHex(spanIdHex)

func init() {
	if err != nil {
		log.Fatal("Fatal error: ", err)
	}
}

func TestSwFromCtx(t *testing.T) {
	sc := trace.SpanContext{}.WithSpanID(spanId).WithTraceFlags(trace.TraceFlags(0x00))

	assert.Equal(t, fmt.Sprintf("%s-00", spanIdHex), SwFromCtx(sc))

	sc = sc.WithTraceFlags(trace.TraceFlags(0x01))
	assert.Equal(t, fmt.Sprintf("%s-01", spanIdHex), SwFromCtx(sc))
}

func TestGetSwTraceState(t *testing.T) {
	tracestate := trace.TraceState{}
	tracestate, err = tracestate.Insert("sw", fmt.Sprintf("%s-01", spanIdHex))
	assert.Nil(t, err)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     spanId,
		TraceFlags: trace.TraceFlags(0x01),
		TraceState: tracestate,
	})

	ts := GetSwTraceState(sc)
	assert.True(t, ts.IsValid())
	assert.Equal(t, spanIdHex, ts.SpanId())
	assert.Equal(t, trace.TraceFlags(0x01), ts.Flags())
}

func TestGetSwTraceStateInvalid(t *testing.T) {
	tracestate := trace.TraceState{}
	tracestate, err = tracestate.Insert("sw", fmt.Sprintf("%s-01", spanIdHex))
	assert.Nil(t, err)
	// Zero'd out trace and span make sc.IsValid() return false, so we should
	// return an invalid trace state
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x00},
		SpanID:     trace.SpanID{0x00},
		TraceFlags: trace.TraceFlags(0x00),
		TraceState: tracestate,
	})

	ts := GetSwTraceState(sc)
	assert.False(t, ts.IsValid())
	assert.Equal(t, "", ts.SpanId())
	assert.Equal(t, trace.TraceFlags(0x00), ts.Flags())
}
func TestParseSwTraceState(t *testing.T) {
	ts := fmt.Sprintf("%s-00", spanIdHex)
	result := parseSwTraceState(ts)
	assert.True(t, result.IsValid())
	assert.Equal(t, spanIdHex, result.SpanId())
	assert.Equal(t, trace.TraceFlags(0x00), result.Flags())

	ts = fmt.Sprintf("%s-01", spanIdHex)
	result = parseSwTraceState(ts)
	assert.True(t, result.IsValid())
	assert.Equal(t, spanIdHex, result.SpanId())
	assert.Equal(t, trace.TraceFlags(0x01), result.Flags())
}

func TestParseInvalidTraceStates(t *testing.T) {
	foo := []string{
		"foo",
		// spanID too long
		"0123456789abcdefa-00",
		// spanID not long enough
		"0123456789abcde-00",
		// spanID not hex
		"g123456789abcdef-00",
		// spanID has uppercase
		"0123456789Abcdef-00",
		// flags too long
		"0123456789abcdef-000",
		// flags not long enough
		"0123456789abcdef-0",
		// flags not hex
		"a123456789abcdef-0g",
		// flags has uppercase
		"0123456789abcdef-0A",
	}

	for _, ts := range foo {
		result := parseSwTraceState(ts)
		assert.False(t, result.IsValid())
		assert.Equal(t, "", result.SpanId())
		assert.Equal(t, trace.TraceFlags(0x00), result.Flags())
	}

}
