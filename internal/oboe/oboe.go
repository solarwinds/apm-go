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
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/rand"
	"github.com/solarwinds/apm-go/internal/w3cfmt"
)

const (
	maxSamplingRate = config.MaxSampleRate
)

// SampleSource enums used by sampling and tracing settings
type SampleSource int

// source of the sample value
const (
	SampleSourceUnset SampleSource = iota - 1
	SampleSourceNone
	SampleSourceFile
	SampleSourceDefault
)

type Oboe interface {
	UpdateSetting(sType int32, layer string, flags []byte, value int64, ttl int64, args map[string][]byte)
	CheckSettingsTimeout()
	GetSetting() (*settings, bool)
	RemoveSetting()
	HasDefaultSetting() bool
	SampleRequest(continued bool, url string, triggerTrace TriggerTraceMode, swState w3cfmt.SwTraceState) SampleDecision
	FlushRateCounts() *metrics.RateCountSummary
	GetTriggerTraceToken() ([]byte, error)
}

func NewOboe() Oboe {
	return &oboe{
		settings: make(map[settingKey]*settings),
	}
}

type oboe struct {
	sync.RWMutex
	settings map[settingKey]*settings
}

var _ Oboe = &oboe{}

func summaryFromSettings(s *settings) *metrics.RateCountSummary {
	regular := s.bucket.FlushRateCounts()
	relaxedTT := s.triggerTraceRelaxedBucket.FlushRateCounts()
	strictTT := s.triggerTraceStrictBucket.FlushRateCounts()
	rcs := []*metrics.RateCounts{regular, relaxedTT, strictTT}

	summary := &metrics.RateCountSummary{
		Sampled:  regular.Sampled(),
		Through:  regular.Through(),
		TtTraced: relaxedTT.Traced() + strictTT.Traced(),
	}

	for _, rc := range rcs {
		summary.Requested += rc.Requested()
		summary.Traced += rc.Traced()
		summary.Limited += rc.Limited()
	}

	return summary
}

// FlushRateCounts collects the request counters values by categories.
func (o *oboe) FlushRateCounts() *metrics.RateCountSummary {
	setting, ok := o.GetSetting()
	if !ok {
		return nil
	}
	return summaryFromSettings(setting)
}

// SampleRequest returns a SampleDecision based on inputs and state of various token buckets
func (o *oboe) SampleRequest(continued bool, url string, triggerTrace TriggerTraceMode, swState w3cfmt.SwTraceState) SampleDecision {
	setting, ok := o.GetSetting()
	if !ok {
		return SampleDecision{false, 0, SampleSourceNone, false, TtSettingsNotAvailable, 0, 0, false}
	}

	var diceRolled, retval, doRateLimiting bool
	sampleRate, flags, source := setting.mergeURLSetting(url)

	// Choose an appropriate bucket
	bucket := setting.bucket
	if triggerTrace == ModeRelaxedTriggerTrace {
		bucket = setting.triggerTraceRelaxedBucket
	} else if triggerTrace == ModeStrictTriggerTrace {
		bucket = setting.triggerTraceStrictBucket
	}

	if triggerTrace.Requested() && !continued {
		sampled := (triggerTrace != ModeInvalidTriggerTrace) && (flags.TriggerTraceEnabled())
		rsp := TtOK

		ret := bucket.count(sampled, false, true)

		if flags.TriggerTraceEnabled() && triggerTrace.Enabled() {
			if !ret {
				rsp = TtRateExceeded
			}
		} else if triggerTrace == ModeInvalidTriggerTrace {
			rsp = ""
		} else {
			if !flags.Enabled() {
				rsp = TtTracingDisabled
			} else {
				rsp = TtTriggerTracingDisabled
			}
		}
		ttCap, ttRate := setting.getTokenBucketSetting(triggerTrace)
		return SampleDecision{ret, -1, SampleSourceUnset, flags.Enabled(), rsp, ttRate, ttCap, diceRolled}
	}

	unsetBucketAndSampleKVs := false
	if !continued {
		// A new request
		if flags&FlagSampleStart != 0 {
			// roll the dice
			diceRolled = true
			retval = shouldSample(sampleRate)
			if retval {
				doRateLimiting = true
			}
		}
	} else if swState.IsValid() {
		if swState.Flags().IsSampled() {
			if flags&FlagSampleThroughAlways != 0 {
				// Conform to liboboe behavior; continue decision would result in a -1 value for the
				// BucketCapacity, BucketRate, SampleRate and SampleSource KVs to indicate "unset".
				unsetBucketAndSampleKVs = true
				retval = true
			} else if flags&FlagSampleThrough != 0 {
				// roll the dice
				diceRolled = true
				retval = shouldSample(sampleRate)
			}
		} else {
			retval = false
		}
	}

	retval = bucket.count(retval, continued, doRateLimiting)

	rsp := TtNotRequested
	if triggerTrace.Requested() {
		rsp = TtIgnored
	}

	var bucketCap, bucketRate float64
	if unsetBucketAndSampleKVs {
		bucketCap, bucketRate, sampleRate, source = -1, -1, -1, SampleSourceUnset
	} else {
		bucketCap, bucketRate = setting.getTokenBucketSetting(ModeTriggerTraceNotPresent)
	}

	return SampleDecision{
		retval,
		sampleRate,
		source,
		flags.Enabled(),
		rsp,
		bucketCap,
		bucketRate,
		diceRolled,
	}
}

func bytesToFloat64(b []byte) (float64, error) {
	if len(b) != 8 {
		return -1, fmt.Errorf("invalid length: %d", len(b))
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
}

func parseFloat64(args map[string][]byte, key string, fb float64) float64 {
	ret := fb
	if c, ok := args[key]; ok {
		v, err := bytesToFloat64(c)
		if err == nil && v >= 0 {
			ret = v
			log.Debugf("parsed %s=%f", key, v)
		} else {
			log.Warningf("parse error: %s=%f err=%v fallback=%f", key, v, err, fb)
		}
	}
	return ret
}

func adjustSampleRate(rate int64) int {
	if rate < 0 {
		log.Debugf("Invalid sample rate: %d", rate)
		return 0
	}

	if rate > maxSamplingRate {
		log.Debugf("Invalid sample rate: %d", rate)
		return maxSamplingRate
	}
	return int(rate)
}

func (o *oboe) UpdateSetting(sType int32, layer string, flags []byte, value int64, ttl int64, args map[string][]byte) {
	ns := newOboeSettings()

	ns.timestamp = time.Now()
	ns.source = settingType(sType).toSampleSource()
	ns.flags = flagStringToBin(string(flags))
	ns.originalFlags = ns.flags
	ns.value = adjustSampleRate(value)
	ns.ttl = ttl
	ns.layer = layer

	ns.TriggerToken = args[constants.KvSignatureKey]

	rate := parseFloat64(args, constants.KvBucketRate, 0)
	capacity := parseFloat64(args, constants.KvBucketCapacity, 0)
	ns.bucket.setRateCap(rate, capacity)

	tRelaxedRate := parseFloat64(args, constants.KvTriggerTraceRelaxedBucketRate, 0)
	tRelaxedCapacity := parseFloat64(args, constants.KvTriggerTraceRelaxedBucketCapacity, 0)
	ns.triggerTraceRelaxedBucket.setRateCap(tRelaxedRate, tRelaxedCapacity)

	tStrictRate := parseFloat64(args, constants.KvTriggerTraceStrictBucketRate, 0)
	tStrictCapacity := parseFloat64(args, constants.KvTriggerTraceStrictBucketCapacity, 0)
	ns.triggerTraceStrictBucket.setRateCap(tStrictRate, tStrictCapacity)

	merged := mergeLocalSetting(ns)

	key := settingKey{
		sType: settingType(sType),
		layer: layer,
	}

	o.Lock()
	o.settings[key] = merged
	o.Unlock()
}

// CheckSettingsTimeout checks and deletes expired settings
func (o *oboe) CheckSettingsTimeout() {
	o.checkSettingsTimeout()
}

func (o *oboe) checkSettingsTimeout() {
	o.Lock()
	defer o.Unlock()

	ss := o.settings
	for k, s := range ss {
		e := s.timestamp.Add(time.Duration(s.ttl) * time.Second)
		if e.Before(time.Now()) {
			delete(ss, k)
		}
	}
}

func (o *oboe) GetSetting() (*settings, bool) {
	o.RLock()
	defer o.RUnlock()

	// always use the default setting
	key := settingKey{
		sType: TypeDefault,
		layer: "",
	}
	if setting, ok := o.settings[key]; ok {
		return setting, true
	}

	return nil, false
}

func (o *oboe) RemoveSetting() {
	o.Lock()
	defer o.Unlock()

	// always use the default setting
	key := settingKey{
		sType: TypeDefault,
		layer: "",
	}

	delete(o.settings, key)
}

func (o *oboe) HasDefaultSetting() bool {
	if _, ok := o.GetSetting(); ok {
		return true
	}
	return false
}

func (o *oboe) GetTriggerTraceToken() ([]byte, error) {
	setting, ok := o.GetSetting()
	if !ok {
		return nil, errors.New("failed to get settings")
	}
	if len(setting.TriggerToken) == 0 {
		return nil, errors.New("no valid signature key found")
	}
	return setting.TriggerToken, nil
}

func shouldSample(sampleRate int) bool {
	return sampleRate == maxSamplingRate || rand.RandIntn(maxSamplingRate) <= sampleRate
}

func flagStringToBin(flagString string) settingFlag {
	flags := settingFlag(0)
	if flagString != "" {
		for _, s := range strings.Split(flagString, ",") {
			switch s {
			case "OVERRIDE":
				flags |= FlagOverride
			case "SAMPLE_START":
				flags |= FlagSampleStart
			case "SAMPLE_THROUGH":
				flags |= FlagSampleThrough
			case "SAMPLE_THROUGH_ALWAYS":
				flags |= FlagSampleThroughAlways
			case "TRIGGER_TRACE":
				flags |= FlagTriggerTrace
			}
		}
	}
	return flags
}
