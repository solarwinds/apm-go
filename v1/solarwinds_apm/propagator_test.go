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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/xtrace"
	"github.com/stretchr/testify/require"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const spanIdHex = "0123456789abcdef"
const traceIdHex = "44444444444444443333333333333333"

var spanId, err1 = trace.SpanIDFromHex(spanIdHex)
var traceId, err2 = trace.TraceIDFromHex(traceIdHex)

func init() {
	for _, err := range []error{err1, err2} {
		if err != nil {
			log.Fatal("Fatal error: ", err)
		}
	}
}

func TestInjectInvalidSpanContext(t *testing.T) {
	sc := trace.SpanContext{}
	assert.False(t, sc.IsValid())
	carrier := propagation.MapCarrier{}
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, "", carrier.Get("tracestate"))
}

func TestInjectNoTracestate(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: 0,
	})
	carrier := propagation.MapCarrier{}
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, fmt.Sprintf("sw=%s-00", spanIdHex), carrier.Get("tracestate"))
}

func TestInjectWithTracestateNoSw(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: 0,
	})
	carrier := propagation.MapCarrier{}
	carrier.Set("tracestate", "other=shouldnotmodify")
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	ts, err := trace.ParseTraceState(carrier.Get("tracestate"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, fmt.Sprintf("%s-00", spanIdHex), ts.Get("sw"))
	assert.Equal(t, "shouldnotmodify", ts.Get("other"))
}

func TestInjectWithTracestatePrevSw(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: 0,
	})
	carrier := propagation.MapCarrier{}
	carrier.Set("tracestate", "sw=012301230-00")
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, fmt.Sprintf("sw=%s-00", spanIdHex), carrier.Get("tracestate"))
}

func TestInjectWithTracestateRemoveXOptsResp(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: 0,
	})
	carrier := propagation.MapCarrier{}
	carrier.Set("tracestate", "sw=012301230-00,xtrace_options_response=foobarbaz")
	p := SolarwindsPropagator{}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	assert.Equal(t, fmt.Sprintf("sw=%s-00", spanIdHex), carrier.Get("tracestate"))
}

func TestExtract(t *testing.T) {
	ctx := context.TODO()
	carrier := propagation.MapCarrier{}

	p := SolarwindsPropagator{}
	newCtx := p.Extract(ctx, carrier)
	assert.Equal(t, ctx, newCtx)
}

func TestExtractXOpts(t *testing.T) {
	carrier := propagation.MapCarrier{
		xtrace.OptionsHeaderName: "foo bar baz",
	}

	p := SolarwindsPropagator{}
	ctx := p.Extract(context.Background(), carrier)
	require.Equal(t, "foo bar baz", ctx.Value(xtrace.OptionsKey))
	require.Nil(t, ctx.Value(xtrace.SignatureKey))
}

func TestExtractXOptsSig(t *testing.T) {
	carrier := propagation.MapCarrier{
		xtrace.OptionsSigHeaderName: "signature",
	}

	p := SolarwindsPropagator{}
	ctx := p.Extract(context.Background(), carrier)
	require.Equal(t, "signature", ctx.Value(xtrace.SignatureKey))
	require.Nil(t, ctx.Value(xtrace.OptionsKey))
}

func TestExtractXOptsAndSig(t *testing.T) {
	carrier := propagation.MapCarrier{
		xtrace.OptionsHeaderName:    "foo bar baz",
		xtrace.OptionsSigHeaderName: "signature",
	}

	p := SolarwindsPropagator{}
	ctx := p.Extract(context.Background(), carrier)
	require.Equal(t, "foo bar baz", ctx.Value(xtrace.OptionsKey))
	require.Equal(t, "signature", ctx.Value(xtrace.SignatureKey))
}

func TestFields(t *testing.T) {
	p := SolarwindsPropagator{}
	assert.Equal(t, []string{"tracestate"}, p.Fields())
}
