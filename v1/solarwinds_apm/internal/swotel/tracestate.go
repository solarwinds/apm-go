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

package swotel

import (
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"strings"
)

const (
	commaSep = "...."
	eqSep    = "####"

	sw = "sw"
)

type InternalTSKey int

const (
	XTraceOptResp InternalTSKey = iota
)

func GetSw(ts trace.TraceState) string {
	return ts.Get(sw)
}

func SetSw(ts trace.TraceState, val string) (trace.TraceState, error) {
	return ts.Insert(sw, val)
}

// INTERNAL STATE

// SetInternalState sets a key in the TraceState, requiring that it be an internal key
func SetInternalState(ts trace.TraceState, key InternalTSKey, val string) (trace.TraceState, error) {
	if k, err := internalKeyStr(key); err != nil {
		return ts, err
	} else {
		val = strings.ReplaceAll(val, ",", commaSep)
		val = strings.ReplaceAll(val, "=", eqSep)
		return ts.Insert(k, val)
	}
}

// GetInternalState retrieves the value from the tracestate, requiring that it be an internal key
func GetInternalState(ts trace.TraceState, key InternalTSKey) (string, error) {
	if k, err := internalKeyStr(key); err != nil {
		return "", err
	} else {
		val := ts.Get(k)
		val = strings.ReplaceAll(val, commaSep, ",")
		val = strings.ReplaceAll(val, eqSep, "=")
		return val, nil
	}
}

// RemoveInternalState removes an internal key from the trace state
func RemoveInternalState(ts trace.TraceState, key InternalTSKey) (trace.TraceState, error) {
	if k, err := internalKeyStr(key); err != nil {
		return ts, err
	} else {
		return ts.Delete(k), nil
	}
}

func internalKeyStr(key InternalTSKey) (string, error) {
	switch key {
	case XTraceOptResp:
		return "xtrace_options_response", nil
	default:
		return "", fmt.Errorf("invalid key: %d", key)
	}
}
