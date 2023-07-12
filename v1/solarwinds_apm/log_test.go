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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestLoggableTraceIDFromContext(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, "00000000000000000000000000000000-0", LoggableTraceIDFromContext(ctx))
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x22},
		SpanID:  trace.SpanID{0x11},
	})

	ctx = trace.ContextWithSpanContext(ctx, sc)
	require.Equal(t, "22000000000000000000000000000000-0", LoggableTraceIDFromContext(ctx))

	sc = trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x33},
		SpanID:     trace.SpanID{0xAA},
		TraceFlags: trace.FlagsSampled,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)
	require.Equal(t, "33000000000000000000000000000000-1", LoggableTraceIDFromContext(ctx))
}
