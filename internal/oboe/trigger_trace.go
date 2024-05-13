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

import "fmt"

// Trigger trace response messages
const (
	TtOK                     = "ok"
	TtRateExceeded           = "rate-exceeded"
	TtTracingDisabled        = "tracing-disabled"
	TtTriggerTracingDisabled = "trigger-tracing-disabled"
	TtNotRequested           = "not-requested"
	TtIgnored                = "ignored"
	TtSettingsNotAvailable   = "settings-not-available"
)

type TriggerTraceMode int

const (
	// ModeTriggerTraceNotPresent means there is no X-Trace-Options header detected,
	// or the X-Trace-Options header is present but trigger_trace flag is not. This
	// indicates that it's a trace for regular sampling.
	ModeTriggerTraceNotPresent TriggerTraceMode = iota

	// ModeInvalidTriggerTrace means X-Trace-Options is detected but no valid trigger-trace
	// flag found, or X-Trace-Options-Signature is present but the authentication is failed.
	ModeInvalidTriggerTrace

	// ModeRelaxedTriggerTrace means X-Trace-Options-Signature is present and valid.
	// The trace will be sampled/limited by the relaxed token bucket.
	ModeRelaxedTriggerTrace

	// ModeStrictTriggerTrace means no X-Trace-Options-Signature is present. The trace
	// will be limited by the strict token bucket.
	ModeStrictTriggerTrace
)

// Enabled indicates whether it's a trigger-trace request
func (tm TriggerTraceMode) Enabled() bool {
	switch tm {
	case ModeTriggerTraceNotPresent, ModeInvalidTriggerTrace:
		return false
	case ModeRelaxedTriggerTrace, ModeStrictTriggerTrace:
		return true
	default:
		panic(fmt.Sprintf("Unhandled trigger trace mode: %x", tm))
	}
}

// Requested indicates whether the user tries to issue a trigger-trace request
// (but may be rejected if the header is illegal)
func (tm TriggerTraceMode) Requested() bool {
	switch tm {
	case ModeTriggerTraceNotPresent:
		return false
	case ModeRelaxedTriggerTrace, ModeStrictTriggerTrace, ModeInvalidTriggerTrace:
		return true
	default:
		panic(fmt.Sprintf("Unhandled trigger trace mode: %x", tm))
	}
}
