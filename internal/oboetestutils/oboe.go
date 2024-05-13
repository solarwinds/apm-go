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

package oboetestutils

import (
	"encoding/binary"
	"github.com/solarwinds/apm-go/internal/constants"
	"math"
)

const TestToken = "TOKEN"
const TypeDefault = 0

func argsToMap(capacity, ratePerSec, tRCap, tRRate, tSCap, tSRate float64,
	metricsFlushInterval, maxTransactions int, token []byte) map[string][]byte {
	args := make(map[string][]byte)

	if capacity > -1 {
		bits := math.Float64bits(capacity)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvBucketCapacity] = bytes
	}
	if ratePerSec > -1 {
		bits := math.Float64bits(ratePerSec)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvBucketRate] = bytes
	}
	if tRCap > -1 {
		bits := math.Float64bits(tRCap)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceRelaxedBucketCapacity] = bytes
	}
	if tRRate > -1 {
		bits := math.Float64bits(tRRate)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceRelaxedBucketRate] = bytes
	}
	if tSCap > -1 {
		bits := math.Float64bits(tSCap)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceStrictBucketCapacity] = bytes
	}
	if tSRate > -1 {
		bits := math.Float64bits(tSRate)
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, bits)
		args[constants.KvTriggerTraceStrictBucketRate] = bytes
	}
	if metricsFlushInterval > -1 {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(metricsFlushInterval))
		args[constants.KvMetricsFlushInterval] = bytes
	}
	if maxTransactions > -1 {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(maxTransactions))
		args[constants.KvMaxTransactions] = bytes
	}

	args[constants.KvSignatureKey] = token

	return args
}

type SettingUpdater interface {
	UpdateSetting(sType int32, layer string, flags []byte, value int64, ttl int64, args map[string][]byte)
}

func AddDefaultSetting(o SettingUpdater) {
	// add default setting with 100% sampling
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte("SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE"),
		1000000, 120, argsToMap(1000000, 1000000, 1000000, 1000000, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddSampleThrough(o SettingUpdater) {
	// add default setting with 100% sampling
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte("SAMPLE_START,SAMPLE_THROUGH,TRIGGER_TRACE"),
		1000000, 120, argsToMap(1000000, 1000000, 1000000, 1000000, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddNoTriggerTrace(o SettingUpdater) {
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte("SAMPLE_START,SAMPLE_THROUGH_ALWAYS"),
		1000000, 120, argsToMap(1000000, 1000000, 0, 0, 0, 0, -1, -1, []byte(TestToken)))
}

func AddTriggerTraceOnly(o SettingUpdater) {
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte("TRIGGER_TRACE"),
		0, 120, argsToMap(0, 0, 1000000, 1000000, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddRelaxedTriggerTraceOnly(o SettingUpdater) {
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte("TRIGGER_TRACE"),
		0, 120, argsToMap(0, 0, 1000000, 1000000, 0, 0, -1, -1, []byte(TestToken)))
}

func AddStrictTriggerTraceOnly(o SettingUpdater) {
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte("TRIGGER_TRACE"),
		0, 120, argsToMap(0, 0, 0, 0, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddLimitedTriggerTrace(o SettingUpdater) {
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte("SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE"),
		1000000, 120, argsToMap(1000000, 1000000, 1, 1, 1, 1, -1, -1, []byte(TestToken)))
}

func AddDisabled(o SettingUpdater) {
	o.UpdateSetting(int32(TypeDefault), "",
		[]byte(""),
		0, 120, argsToMap(0, 0, 1, 1, 1, 1, -1, -1, []byte(TestToken)))
}
