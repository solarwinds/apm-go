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
	"encoding/hex"
	"fmt"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/swotel"
	"log/slog"
	"regexp"

	"go.opentelemetry.io/otel/trace"
)

var invalidSwTraceState = SwTraceState{
	isValid: false,
	spanId:  "",
	flags:   0,
}

// Note: We only accept lowercase hex, and therefore cannot use `[[:xdigit:]]`
var swTraceStateRegex = regexp.MustCompile(`^([0-9a-f]{16})-([0-9a-f]{2})$`)

func SwFromCtx(sc trace.SpanContext) string {
	spanID := sc.SpanID()
	traceFlags := sc.TraceFlags()
	return fmt.Sprintf("%x-%x", spanID[:], []byte{byte(traceFlags)})
}

func GetSwTraceState(ctx trace.SpanContext) SwTraceState {
	if ctx.IsValid() {
		tracestate := ctx.TraceState()
		swVal := swotel.GetSw(tracestate)
		slog.Info("getSw", "sw", swVal)
		return ParseSwTraceState(swVal)
	}
	return invalidSwTraceState
}

func ParseSwTraceState(s string) SwTraceState {
	matches := swTraceStateRegex.FindStringSubmatch(s)
	if matches != nil {
		flags, err := hex.DecodeString(matches[2])
		if err != nil {
			log.Debug("Could not decode hex!", matches[2])
			matches = nil
		}
		return SwTraceState{isValid: true, spanId: matches[1], flags: trace.TraceFlags(flags[0])}
	}

	return invalidSwTraceState
}

type SwTraceState struct {
	isValid bool
	spanId  string
	flags   trace.TraceFlags
}

func (s SwTraceState) IsValid() bool {
	return s.isValid
}

func (s SwTraceState) SpanId() string {
	return s.spanId
}

func (s SwTraceState) Flags() trace.TraceFlags {
	return s.flags
}
