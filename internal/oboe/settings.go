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

import (
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"time"
)

type settings struct {
	timestamp time.Time
	// the flags which may be modified through merging local settings.
	flags settingFlag
	// the original flags retrieved from the remote collector.
	originalFlags settingFlag
	// The sample rate. It could be the original value got from remote server
	// or a new value after negotiating with local config
	value int
	// The sample source after negotiating with local config
	source                    SampleSource
	ttl                       int64
	layer                     string
	TriggerToken              []byte
	bucket                    *tokenBucket
	triggerTraceRelaxedBucket *tokenBucket
	triggerTraceStrictBucket  *tokenBucket
}

func (s *settings) hasOverrideFlag() bool {
	return s.originalFlags&FLAG_OVERRIDE != 0
}
func newOboeSettings() *settings {
	return &settings{
		// The global token bucket. Trace decisions of all the requests are controlled
		// by this single bucket.
		//
		// The rate and capacity will be initialized by the values fetched from the remote
		// server, therefore it's initialized with only the default values.
		bucket: &tokenBucket{},
		// The token bucket exclusively for trigger trace from authenticated clients
		triggerTraceRelaxedBucket: &tokenBucket{},
		// The token bucket exclusively for trigger trace from unauthenticated clients
		triggerTraceStrictBucket: &tokenBucket{},
	}
}

// mergeLocalSetting follow the predefined precedence to decide which one to
// pick from: either the local configs or the remote ones, or the combination.
//
// Note: This function modifies the argument in place.
func mergeLocalSetting(remote *settings) *settings {
	if remote.hasOverrideFlag() && config.SamplingConfigured() {
		// Choose the lower sample rate and merge the flags
		if remote.value > config.GetSampleRate() {
			remote.value = config.GetSampleRate()
			remote.source = SAMPLE_SOURCE_FILE
		}
		remote.flags &= NewTracingMode(config.GetTracingMode()).toFlags()
	} else if config.SamplingConfigured() {
		// Use local sample rate and tracing mode config
		remote.value = config.GetSampleRate()
		remote.flags = NewTracingMode(config.GetTracingMode()).toFlags()
		remote.source = SAMPLE_SOURCE_FILE
	}

	if !config.GetTriggerTrace() {
		remote.flags = remote.flags &^ (1 << FlagTriggerTraceOffset)
	}
	return remote
}

// mergeURLSetting merges the service level setting (merged from remote and local
// settings) and the per-URL sampling flags, if any.
func (s *settings) mergeURLSetting(url string) (int, settingFlag, SampleSource) {
	if url == "" {
		return s.value, s.flags, s.source
	}

	urlTracingMode := urls.GetTracingMode(url)
	if urlTracingMode.isUnknown() {
		return s.value, s.flags, s.source
	}

	flags := urlTracingMode.toFlags()
	source := SAMPLE_SOURCE_FILE

	if s.hasOverrideFlag() {
		flags &= s.originalFlags
	}

	return s.value, flags, source
}

func (s *settings) getTokenBucketSetting(ttMode TriggerTraceMode) (capacity float64, rate float64) {
	var bucket *tokenBucket

	switch ttMode {
	case ModeRelaxedTriggerTrace:
		bucket = s.triggerTraceRelaxedBucket
	case ModeStrictTriggerTrace:
		bucket = s.triggerTraceStrictBucket
	case ModeTriggerTraceNotPresent, ModeInvalidTriggerTrace:
		bucket = s.bucket
	default:
		log.Warningf("Could not determine token bucket setting for invalid TriggerTraceMode: %#v", ttMode)
		return 0, 0
	}

	return bucket.capacity, bucket.ratePerSec
}

// The identifying keys for a setting
type settingKey struct {
	sType settingType
	layer string
}
type settingType int
type settingFlag uint16

// setting types
const (
	TYPE_DEFAULT settingType = iota // default setting which serves as a fallback if no other settings found
	TYPE_LAYER                      // layer specific settings
)

// setting flags offset
const (
	FlagInvalidOffset = iota
	FlagOverrideOffset
	FlagSampleStartOffset
	FlagSampleThroughOffset
	FlagSampleThroughAlwaysOffset
	FlagTriggerTraceOffset
)

// setting flags
const (
	FLAG_OK                    settingFlag = 0x0
	FLAG_INVALID               settingFlag = 1 << FlagInvalidOffset
	FLAG_OVERRIDE              settingFlag = 1 << FlagOverrideOffset
	FLAG_SAMPLE_START          settingFlag = 1 << FlagSampleStartOffset
	FLAG_SAMPLE_THROUGH        settingFlag = 1 << FlagSampleThroughOffset
	FLAG_SAMPLE_THROUGH_ALWAYS settingFlag = 1 << FlagSampleThroughAlwaysOffset
	FLAG_TRIGGER_TRACE         settingFlag = 1 << FlagTriggerTraceOffset
)

// Enabled returns if the trace is enabled or not.
func (f settingFlag) Enabled() bool {
	return f&(FLAG_SAMPLE_START|FLAG_SAMPLE_THROUGH_ALWAYS) != 0
}

// TriggerTraceEnabled returns if the trigger trace is enabled
func (f settingFlag) TriggerTraceEnabled() bool {
	return f&FLAG_TRIGGER_TRACE != 0
}

func (st settingType) toSampleSource() SampleSource {
	var source SampleSource
	switch st {
	case TYPE_DEFAULT:
		source = SAMPLE_SOURCE_DEFAULT
	case TYPE_LAYER:
		source = SAMPLE_SOURCE_LAYER
	default:
		source = SAMPLE_SOURCE_NONE
	}
	return source
}
