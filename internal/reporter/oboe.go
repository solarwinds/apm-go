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

package reporter

import (
	"encoding/binary"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/config"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/metrics"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/rand"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/swotel/semconv"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/utils"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/w3cfmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	maxSamplingRate = config.MaxSampleRate
)

// enums used by sampling and tracing settings
type SampleSource int

// source of the sample value
const (
	SAMPLE_SOURCE_UNSET SampleSource = iota - 1
	SAMPLE_SOURCE_NONE
	SAMPLE_SOURCE_FILE
	SAMPLE_SOURCE_DEFAULT
	SAMPLE_SOURCE_LAYER
)

// Current settings configuration
type oboeSettingsCfg struct {
	settings map[oboeSettingKey]*oboeSettings
	lock     sync.RWMutex
}

// FlushRateCounts collects the request counters values by categories.
func FlushRateCounts() map[string]*metrics.RateCounts {
	setting, ok := getSetting()
	if !ok {
		return nil
	}
	rcs := make(map[string]*metrics.RateCounts)
	rcs[metrics.RCRegular] = setting.bucket.FlushRateCounts()
	rcs[metrics.RCRelaxedTriggerTrace] = setting.triggerTraceRelaxedBucket.FlushRateCounts()
	rcs[metrics.RCStrictTriggerTrace] = setting.triggerTraceStrictBucket.FlushRateCounts()

	return rcs
}

type oboeSettings struct {
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
	triggerToken              []byte
	bucket                    *tokenBucket
	triggerTraceRelaxedBucket *tokenBucket
	triggerTraceStrictBucket  *tokenBucket
}

func (s *oboeSettings) hasOverrideFlag() bool {
	return s.originalFlags&FLAG_OVERRIDE != 0
}

func newOboeSettings() *oboeSettings {
	return &oboeSettings{
		bucket:                    globalTokenBucket,
		triggerTraceRelaxedBucket: triggerTraceRelaxedBucket,
		triggerTraceStrictBucket:  triggerTraceStrictBucket,
	}
}

// token bucket
type tokenBucket struct {
	ratePerSec float64
	capacity   float64
	available  float64
	last       time.Time
	lock       sync.Mutex
	metrics.RateCounts
}

func (b *tokenBucket) reset() {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.ratePerSec = 0
	b.capacity = 0
	b.available = 0
	b.last = time.Time{}
}

func (b *tokenBucket) setRateCap(rate, cap float64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.ratePerSec = rate
	b.capacity = cap

	if b.available > b.capacity {
		b.available = b.capacity
	}
}

// The identifying keys for a setting
type oboeSettingKey struct {
	sType settingType
	layer string
}

// Global configuration settings
var globalSettingsCfg = &oboeSettingsCfg{
	settings: make(map[oboeSettingKey]*oboeSettings),
}

// The global token bucket. Trace decisions of all the requests are controlled
// by this single bucket.
//
// The rate and capacity will be initialized by the values fetched from the remote
// server, therefore it's initialized with only the default values.
var globalTokenBucket = &tokenBucket{}

// The token bucket exclusively for trigger trace from authenticated clients
var triggerTraceRelaxedBucket = &tokenBucket{}

// The token bucket exclusively for trigger trace from unauthenticated clients
var triggerTraceStrictBucket = &tokenBucket{}

func createInitMessage(tid trace.TraceID, r *resource.Resource) Event {
	evt := NewEventWithRandomOpID(tid, time.Now())
	evt.SetLabel(LabelUnset)
	for _, kv := range r.Attributes() {
		if kv.Key != semconv.ServiceNameKey {
			evt.AddKV(kv)
		}
	}

	evt.AddKVs([]attribute.KeyValue{
		attribute.Bool("__Init", true),
		attribute.String("APM.Version", utils.Version()),
	})
	return evt
}

func sendInitMessage(r *resource.Resource) {
	if Closed() {
		log.Info(errors.Wrap(ErrReporterIsClosed, "send init message"))
		return
	}
	tid := trace.TraceID{0}
	rand.Random(tid[:])
	evt := createInitMessage(tid, r)
	if err := ReportStatus(evt); err != nil {
		log.Error("could not send init message", err)
	}
}

func (b *tokenBucket) count(sampled, hasMetadata, rateLimit bool) bool {
	b.RequestedInc()

	if !hasMetadata {
		b.SampledInc()
	}

	if !sampled {
		return sampled
	}

	if rateLimit {
		if ok := b.consume(1); !ok {
			b.LimitedInc()
			return false
		}
	}

	if hasMetadata {
		b.ThroughInc()
	}
	b.TracedInc()
	return sampled
}

func (b *tokenBucket) consume(size float64) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.update(time.Now())
	if b.available >= size {
		b.available -= size
		return true
	}
	return false
}

func (b *tokenBucket) update(now time.Time) {
	if b.available < b.capacity { // room for more tokens?
		delta := now.Sub(b.last) // calculate duration since last check
		b.last = now             // update time of last check
		if delta <= 0 {          // return if no delta or time went "backwards"
			return
		}
		newTokens := b.ratePerSec * delta.Seconds()               // # tokens generated since last check
		b.available = math.Min(b.capacity, b.available+newTokens) // add new tokens to bucket, but don't overfill
	}
}

type SampleDecision struct {
	trace  bool
	rate   int
	source SampleSource
	// if the request is disabled from tracing in a per-transaction level or for
	// the entire service.
	enabled       bool
	xTraceOptsRsp string
	bucketCap     float64
	bucketRate    float64

	diceRolled bool
}

func (s SampleDecision) Trace() bool {
	return s.trace
}

func (s SampleDecision) XTraceOptsRsp() string {
	return s.xTraceOptsRsp
}

func (s SampleDecision) Enabled() bool {
	return s.enabled
}

func (s SampleDecision) BucketCapacity() float64 {
	return s.bucketCap
}

func (s SampleDecision) BucketCapacityStr() string {
	return floatToStr(s.BucketCapacity())
}

func (s SampleDecision) BucketRate() float64 {
	return s.bucketRate
}

func (s SampleDecision) BucketRateStr() string {
	return floatToStr(s.BucketRate())
}

func (s SampleDecision) SampleRate() int {
	return s.rate
}

func (s SampleDecision) SampleSource() SampleSource {
	return s.source
}

func floatToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

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

// Trigger trace response messages
const (
	ttOK                     = "ok"
	ttRateExceeded           = "rate-exceeded"
	ttTracingDisabled        = "tracing-disabled"
	ttTriggerTracingDisabled = "trigger-tracing-disabled"
	ttNotRequested           = "not-requested"
	ttIgnored                = "ignored"
	ttSettingsNotAvailable   = "settings-not-available"
	ttEmpty                  = ""
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

func oboeSampleRequest(continued bool, url string, triggerTrace TriggerTraceMode, swState w3cfmt.SwTraceState) SampleDecision {
	if usingTestReporter {
		if r, ok := globalReporter.(*TestReporter); ok {
			if !r.UseSettings {
				return SampleDecision{r.ShouldTrace, 0, SAMPLE_SOURCE_NONE, true, ttEmpty, 0, 0, false} // trace tests
			}
		}
	}

	var setting *oboeSettings
	var ok bool
	diceRolled := false
	if setting, ok = getSetting(); !ok {
		return SampleDecision{false, 0, SAMPLE_SOURCE_NONE, false, ttSettingsNotAvailable, 0, 0, diceRolled}
	}

	retval := false
	doRateLimiting := false

	sampleRate, flags, source := mergeURLSetting(setting, url)

	// Choose an appropriate bucket
	bucket := setting.bucket
	if triggerTrace == ModeRelaxedTriggerTrace {
		bucket = setting.triggerTraceRelaxedBucket
	} else if triggerTrace == ModeStrictTriggerTrace {
		bucket = setting.triggerTraceStrictBucket
	}

	if triggerTrace.Requested() && !continued {
		sampled := (triggerTrace != ModeInvalidTriggerTrace) && (flags.TriggerTraceEnabled())
		rsp := ttOK

		ret := bucket.count(sampled, false, true)

		if flags.TriggerTraceEnabled() && triggerTrace.Enabled() {
			if !ret {
				rsp = ttRateExceeded
			}
		} else if triggerTrace == ModeInvalidTriggerTrace {
			rsp = ""
		} else {
			if !flags.Enabled() {
				rsp = ttTracingDisabled
			} else {
				rsp = ttTriggerTracingDisabled
			}
		}
		ttCap, ttRate := getTokenBucketSetting(setting, triggerTrace)
		return SampleDecision{ret, -1, SAMPLE_SOURCE_UNSET, flags.Enabled(), rsp, ttRate, ttCap, diceRolled}
	}

	unsetBucketAndSampleKVs := false
	if !continued {
		// A new request
		if flags&FLAG_SAMPLE_START != 0 {
			// roll the dice
			diceRolled = true
			retval = shouldSample(sampleRate)
			if retval {
				doRateLimiting = true
			}
		}
	} else if swState.IsValid() {
		if swState.Flags().IsSampled() {
			if flags&FLAG_SAMPLE_THROUGH_ALWAYS != 0 {
				// Conform to liboboe behavior; continue decision would result in a -1 value for the
				// BucketCapacity, BucketRate, SampleRate and SampleSource KVs to indicate "unset".
				unsetBucketAndSampleKVs = true
				retval = true
			} else if flags&FLAG_SAMPLE_THROUGH != 0 {
				// roll the dice
				diceRolled = true
				retval = shouldSample(sampleRate)
			}
		} else {
			retval = false
		}
	}

	retval = bucket.count(retval, continued, doRateLimiting)

	rsp := ttNotRequested
	if triggerTrace.Requested() {
		rsp = ttIgnored
	}

	var bucketCap, bucketRate float64
	if unsetBucketAndSampleKVs {
		bucketCap, bucketRate, sampleRate, source = -1, -1, -1, SAMPLE_SOURCE_UNSET
	} else {
		bucketCap, bucketRate = getTokenBucketSetting(setting, ModeTriggerTraceNotPresent)
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

func getTokenBucketSetting(setting *oboeSettings, ttMode TriggerTraceMode) (capacity float64, rate float64) {
	var bucket *tokenBucket

	switch ttMode {
	case ModeRelaxedTriggerTrace:
		bucket = setting.triggerTraceRelaxedBucket
	case ModeStrictTriggerTrace:
		bucket = setting.triggerTraceStrictBucket
	case ModeTriggerTraceNotPresent, ModeInvalidTriggerTrace:
		bucket = setting.bucket
	default:
		log.Warningf("Could not determine token bucket setting for invalid TriggerTraceMode: %#v", ttMode)
		return 0, 0
	}

	return bucket.capacity, bucket.ratePerSec
}

func bytesToFloat64(b []byte) (float64, error) {
	if len(b) != 8 {
		return -1, fmt.Errorf("invalid length: %d", len(b))
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
}

func bytesToInt32(b []byte) (int32, error) {
	if len(b) != 4 {
		return -1, fmt.Errorf("invalid length: %d", len(b))
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
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

func parseInt32(args map[string][]byte, key string, fb int32) int32 {
	ret := fb
	if c, ok := args[key]; ok {
		v, err := bytesToInt32(c)
		if err == nil && v >= 0 {
			ret = v
			log.Debugf("parsed %s=%d", key, v)
		} else {
			log.Warningf("parse error: %s=%d err=%v fallback=%d", key, v, err, fb)
		}
	}
	return ret
}

// mergeLocalSetting follow the predefined precedence to decide which one to
// pick from: either the local configs or the remote ones, or the combination.
//
// Note: This function modifies the argument in place.
func mergeLocalSetting(remote *oboeSettings) *oboeSettings {
	if remote.hasOverrideFlag() && config.SamplingConfigured() {
		// Choose the lower sample rate and merge the flags
		if remote.value > config.GetSampleRate() {
			remote.value = config.GetSampleRate()
			remote.source = SAMPLE_SOURCE_FILE
		}
		remote.flags &= newTracingMode(config.GetTracingMode()).toFlags()
	} else if config.SamplingConfigured() {
		// Use local sample rate and tracing mode config
		remote.value = config.GetSampleRate()
		remote.flags = newTracingMode(config.GetTracingMode()).toFlags()
		remote.source = SAMPLE_SOURCE_FILE
	}

	if !config.GetTriggerTrace() {
		remote.flags = remote.flags &^ (1 << FlagTriggerTraceOffset)
	}
	return remote
}

// mergeURLSetting merges the service level setting (merged from remote and local
// settings) and the per-URL sampling flags, if any.
func mergeURLSetting(setting *oboeSettings, url string) (int, settingFlag, SampleSource) {
	if url == "" {
		return setting.value, setting.flags, setting.source
	}

	urlTracingMode := urls.getTracingMode(url)
	if urlTracingMode.isUnknown() {
		return setting.value, setting.flags, setting.source
	}

	flags := urlTracingMode.toFlags()
	source := SAMPLE_SOURCE_FILE

	if setting.hasOverrideFlag() {
		flags &= setting.originalFlags
	}

	return setting.value, flags, source
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

func updateSetting(sType int32, layer string, flags []byte, value int64, ttl int64, args map[string][]byte) {
	ns := newOboeSettings()

	ns.timestamp = time.Now()
	ns.source = settingType(sType).toSampleSource()
	ns.flags = flagStringToBin(string(flags))
	ns.originalFlags = ns.flags
	ns.value = adjustSampleRate(value)
	ns.ttl = ttl
	ns.layer = layer

	ns.triggerToken = args[kvSignatureKey]

	rate := parseFloat64(args, kvBucketRate, 0)
	capacity := parseFloat64(args, kvBucketCapacity, 0)
	ns.bucket.setRateCap(rate, capacity)

	tRelaxedRate := parseFloat64(args, kvTriggerTraceRelaxedBucketRate, 0)
	tRelaxedCapacity := parseFloat64(args, kvTriggerTraceRelaxedBucketCapacity, 0)
	ns.triggerTraceRelaxedBucket.setRateCap(tRelaxedRate, tRelaxedCapacity)

	tStrictRate := parseFloat64(args, kvTriggerTraceStrictBucketRate, 0)
	tStrictCapacity := parseFloat64(args, kvTriggerTraceStrictBucketCapacity, 0)
	ns.triggerTraceStrictBucket.setRateCap(tStrictRate, tStrictCapacity)

	merged := mergeLocalSetting(ns)

	key := oboeSettingKey{
		sType: settingType(sType),
		layer: layer,
	}

	globalSettingsCfg.lock.Lock()
	globalSettingsCfg.settings[key] = merged
	globalSettingsCfg.lock.Unlock()
}

// Used for tests only
func resetSettings() {
	FlushRateCounts()

	globalSettingsCfg.lock.Lock()
	defer globalSettingsCfg.lock.Unlock()
	globalSettingsCfg.settings = make(map[oboeSettingKey]*oboeSettings)
	globalTokenBucket.reset()
}

// OboeCheckSettingsTimeout checks and deletes expired settings
func OboeCheckSettingsTimeout() {
	globalSettingsCfg.checkSettingsTimeout()
}

func (sc *oboeSettingsCfg) checkSettingsTimeout() {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	ss := sc.settings
	for k, s := range ss {
		e := s.timestamp.Add(time.Duration(s.ttl) * time.Second)
		if e.Before(time.Now()) {
			delete(ss, k)
		}
	}
}

func getSetting() (*oboeSettings, bool) {
	globalSettingsCfg.lock.RLock()
	defer globalSettingsCfg.lock.RUnlock()

	// for now only look up the default settings
	key := oboeSettingKey{
		sType: TYPE_DEFAULT,
		layer: "",
	}
	if setting, ok := globalSettingsCfg.settings[key]; ok {
		return setting, true
	}

	return nil, false
}

func removeSetting() {
	globalSettingsCfg.lock.Lock()
	defer globalSettingsCfg.lock.Unlock()

	key := oboeSettingKey{
		sType: TYPE_DEFAULT,
		layer: "",
	}

	delete(globalSettingsCfg.settings, key)
}

func hasDefaultSetting() bool {
	if _, ok := getSetting(); ok {
		return true
	}
	return false
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
				flags |= FLAG_OVERRIDE
			case "SAMPLE_START":
				flags |= FLAG_SAMPLE_START
			case "SAMPLE_THROUGH":
				flags |= FLAG_SAMPLE_THROUGH
			case "SAMPLE_THROUGH_ALWAYS":
				flags |= FLAG_SAMPLE_THROUGH_ALWAYS
			case "TRIGGER_TRACE":
				flags |= FLAG_TRIGGER_TRACE
			}
		}
	}
	return flags
}

// tracing mode
type tracingMode int

// tracing modes
const (
	TraceDisabled tracingMode = iota // disable tracing, will neither start nor continue traces
	TraceEnabled                     // perform sampling every inbound request for tracing
	TraceUnknown                     // for cache purpose only
)

// newTracingMode creates a tracing mode object from a string
func newTracingMode(mode config.TracingMode) tracingMode {
	switch mode {
	case config.DisabledTracingMode:
		return TraceDisabled
	case config.EnabledTracingMode:
		return TraceEnabled
	default:
	}
	return TraceUnknown
}

func (tm tracingMode) isUnknown() bool {
	return tm == TraceUnknown
}

func (tm tracingMode) toFlags() settingFlag {
	switch tm {
	case TraceEnabled:
		return FLAG_SAMPLE_START | FLAG_SAMPLE_THROUGH_ALWAYS | FLAG_TRIGGER_TRACE
	case TraceDisabled:
	default:
	}
	return FLAG_OK
}

func (tm tracingMode) ToString() string {
	switch tm {
	case TraceEnabled:
		return string(config.EnabledTracingMode)
	case TraceDisabled:
		return string(config.DisabledTracingMode)
	default:
		return string(config.UnknownTracingMode)
	}
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
