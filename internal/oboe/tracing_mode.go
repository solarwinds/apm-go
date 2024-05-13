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

import "github.com/solarwinds/apm-go/internal/config"

type TracingMode int

const (
	TraceDisabled TracingMode = iota // disable tracing, will neither start nor continue traces
	TraceEnabled                     // perform sampling every inbound request for tracing
	TraceUnknown                     // for cache purpose only
)

// NewTracingMode creates a tracing mode object from a string
func NewTracingMode(mode config.TracingMode) TracingMode {
	switch mode {
	case config.DisabledTracingMode:
		return TraceDisabled
	case config.EnabledTracingMode:
		return TraceEnabled
	default:
	}
	return TraceUnknown
}

func (tm TracingMode) isUnknown() bool {
	return tm == TraceUnknown
}

func (tm TracingMode) toFlags() settingFlag {
	switch tm {
	case TraceEnabled:
		return FlagSampleStart | FlagSampleThroughAlways | FlagTriggerTrace
	case TraceDisabled:
	default:
	}
	return FlagOk
}

func (tm TracingMode) ToString() string {
	switch tm {
	case TraceEnabled:
		return string(config.EnabledTracingMode)
	case TraceDisabled:
		return string(config.DisabledTracingMode)
	default:
		return string(config.UnknownTracingMode)
	}
}
