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
	"go.opentelemetry.io/otel/trace"
)

// LoggableTraceIDFromContext Returns a loggable trace ID from the given
// context.Context for log injection, or an empty string if the trace
// is invalid
func LoggableTraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	return LoggableTraceIDFromSpanContext(sc)
}

func LoggableTraceIDFromSpanContext(ctx trace.SpanContext) string {
	if !ctx.IsValid() {
		return ""
	}
	tid := ctx.TraceID().String()
	sampled := "0"
	if ctx.IsSampled() {
		sampled = "1"
	}
	return fmt.Sprintf("%s-%s", tid, sampled)
}
