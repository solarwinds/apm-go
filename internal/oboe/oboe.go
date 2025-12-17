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
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/solarwinds/apm-go/internal/config"
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

type SettingsUpdateArgs struct {
	Flags                        string
	Value                        int64
	Ttl                          time.Duration
	TriggerToken                 []byte
	BucketCapacity               float64
	BucketRate                   float64
	MetricsFlushInterval         int
	TriggerRelaxedBucketCapacity float64
	TriggerRelaxedBucketRate     float64
	TriggerStrictBucketCapacity  float64
	TriggerStrictBucketRate      float64
}

type Oboe interface {
	UpdateSetting(arg SettingsUpdateArgs)
	CheckSettingsTimeout()
	GetSetting() *settings
	RemoveSetting()
	HasDefaultSetting() bool
	SampleRequest(continued bool, url string, triggerTrace TriggerTraceMode, swState w3cfmt.SwTraceState) SampleDecision
	FlushRateCounts() *metrics.RateCountSummary
	GetTriggerTraceToken() ([]byte, error)
	RegisterOtelSampleRateMetrics(mp metric.MeterProvider) error
}

func NewOboe() Oboe {
	return &oboe{}
}

type oboe struct {
	settings atomic.Pointer[settings]
}

var _ Oboe = &oboe{}

func (o *oboe) RegisterOtelSampleRateMetrics(mp metric.MeterProvider) error {
	meter := mp.Meter("sw.apm.sampling.metrics")
	traceCount, err := meter.Int64ObservableGauge("trace.service.tracecount")
	if err != nil {
		return err
	}
	sampleCount, err := meter.Int64ObservableGauge("trace.service.samplecount")
	if err != nil {
		return err
	}
	requestCount, err := meter.Int64ObservableGauge("trace.service.request_count")
	if err != nil {
		return err
	}
	tokenBucketExhaustionCount, err := meter.Int64ObservableGauge("trace.service.tokenbucket_exhaustion_count")
	if err != nil {
		return err
	}
	throughTraceCount, err := meter.Int64ObservableGauge("trace.service.through_trace_count")
	if err != nil {
		return err
	}
	triggeredTraceCount, err := meter.Int64ObservableGauge("trace.service.triggered_trace_count")
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(
		func(_ context.Context, obs metric.Observer) error {
			if rateCounts := o.FlushRateCounts(); rateCounts != nil {
				obs.ObserveInt64(traceCount, rateCounts.Traced)
				obs.ObserveInt64(sampleCount, rateCounts.Sampled)
				obs.ObserveInt64(requestCount, rateCounts.Requested)
				obs.ObserveInt64(tokenBucketExhaustionCount, rateCounts.Limited)
				obs.ObserveInt64(throughTraceCount, rateCounts.Through)
				obs.ObserveInt64(triggeredTraceCount, rateCounts.TtTraced)
			}
			return nil
		},
		traceCount,
		sampleCount,
		requestCount,
		tokenBucketExhaustionCount,
		throughTraceCount,
		triggeredTraceCount,
	)
	return err
}

// FlushRateCounts collects the request counters values by categories.
func (o *oboe) FlushRateCounts() *metrics.RateCountSummary {
	s := o.GetSetting()
	if s == nil {
		return nil
	}
	counts := metrics.RatesAggregator().FlushRateCounts()

	return &metrics.RateCountSummary{
		Sampled:   counts.Sampled(),
		Through:   counts.Through(),
		Requested: counts.Requested(),
		Traced:    counts.Traced(),
		Limited:   counts.Limited(),
		TtTraced:  counts.TriggerTrace(),
	}
}

// SampleRequest returns a SampleDecision based on inputs and state of various token buckets
func (o *oboe) SampleRequest(continued bool, url string, triggerTrace TriggerTraceMode, swState w3cfmt.SwTraceState) SampleDecision {
	setting := o.GetSetting()
	if setting == nil {
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

		ret := bucket.count(sampled, false, true, true)

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

	retval = bucket.count(retval, continued, doRateLimiting, false)

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

func (o *oboe) UpdateSetting(arg SettingsUpdateArgs) {
	ns := newOboeSettings()

	ns.timestamp = time.Now()
	ns.source = SampleSourceDefault
	ns.flags = flagStringToBin(arg.Flags)
	ns.originalFlags = ns.flags
	ns.value = adjustSampleRate(arg.Value)
	ns.ttl = arg.Ttl
	ns.TriggerToken = arg.TriggerToken

	ns.bucket.setRateCap(arg.BucketRate, arg.BucketCapacity)
	ns.triggerTraceRelaxedBucket.setRateCap(arg.TriggerRelaxedBucketRate, arg.TriggerRelaxedBucketCapacity)
	ns.triggerTraceStrictBucket.setRateCap(arg.TriggerStrictBucketRate, arg.TriggerStrictBucketCapacity)

	ns.MergeLocalSetting()
	o.settings.Store(ns)
}

// CheckSettingsTimeout checks and deletes expired settings
func (o *oboe) CheckSettingsTimeout() {
	o.checkSettingsTimeout()
}

func (o *oboe) checkSettingsTimeout() {
	s := o.settings.Load()
	if s == nil {
		log.Debug("checkSettingsTimeout: No settings")
		return
	}
	e := s.timestamp.Add(s.ttl)
	log.Debugf("checkSettingsTimeout: ttl: %s, timestamp: %s, boundary: %s", s.ttl, s.timestamp, e)
	if e.Before(time.Now()) {
		log.Debugf("checkSettingsTimeout: ttl exceeded, expiring settings")
		o.settings.Store(nil)
	}
}

func (o *oboe) GetSetting() *settings {
	return o.settings.Load()
}

func (o *oboe) RemoveSetting() {
	o.settings.Store(nil)
}

func (o *oboe) HasDefaultSetting() bool {
	return o.settings.Load() != nil
}

func (o *oboe) GetTriggerTraceToken() ([]byte, error) {
	setting := o.GetSetting()
	if setting == nil {
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

func flagStringToBin(flags string) settingFlag {
	result := settingFlag(0)
	if flags != "" {
		for _, s := range strings.Split(flags, ",") {
			switch s {
			case "OVERRIDE":
				result |= FlagOverride
			case "SAMPLE_START":
				result |= FlagSampleStart
			case "SAMPLE_THROUGH":
				result |= FlagSampleThrough
			case "SAMPLE_THROUGH_ALWAYS":
				result |= FlagSampleThroughAlways
			case "TRIGGER_TRACE":
				result |= FlagTriggerTrace
			}
		}
	}
	return result
}
