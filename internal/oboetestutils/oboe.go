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

import "github.com/solarwinds/apm-go/internal/utils"

const TestToken = "TOKEN"
const TypeDefault = 0

type SettingUpdater interface {
	UpdateSetting(layer string, flags []byte, value int64, ttl int64, args map[string][]byte)
}

func AddDefaultSetting(o SettingUpdater) {
	// add default setting with 100% sampling
	o.UpdateSetting("",
		[]byte("SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE"),
		1000000, 120, utils.ArgsToMap(1000000, 1000000, 1000000, 1000000, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddSampleThrough(o SettingUpdater) {
	// add default setting with 100% sampling
	o.UpdateSetting("",
		[]byte("SAMPLE_START,SAMPLE_THROUGH,TRIGGER_TRACE"),
		1000000, 120, utils.ArgsToMap(1000000, 1000000, 1000000, 1000000, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddNoTriggerTrace(o SettingUpdater) {
	o.UpdateSetting("",
		[]byte("SAMPLE_START,SAMPLE_THROUGH_ALWAYS"),
		1000000, 120, utils.ArgsToMap(1000000, 1000000, 0, 0, 0, 0, -1, -1, []byte(TestToken)))
}

func AddTriggerTraceOnly(o SettingUpdater) {
	o.UpdateSetting("",
		[]byte("TRIGGER_TRACE"),
		0, 120, utils.ArgsToMap(0, 0, 1000000, 1000000, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddRelaxedTriggerTraceOnly(o SettingUpdater) {
	o.UpdateSetting("",
		[]byte("TRIGGER_TRACE"),
		0, 120, utils.ArgsToMap(0, 0, 1000000, 1000000, 0, 0, -1, -1, []byte(TestToken)))
}

func AddStrictTriggerTraceOnly(o SettingUpdater) {
	o.UpdateSetting("",
		[]byte("TRIGGER_TRACE"),
		0, 120, utils.ArgsToMap(0, 0, 0, 0, 1000000, 1000000, -1, -1, []byte(TestToken)))
}

func AddLimitedTriggerTrace(o SettingUpdater) {
	o.UpdateSetting("",
		[]byte("SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE"),
		1000000, 120, utils.ArgsToMap(1000000, 1000000, 1, 1, 1, 1, -1, -1, []byte(TestToken)))
}

func AddDisabled(o SettingUpdater) {
	o.UpdateSetting("",
		[]byte(""),
		0, 120, utils.ArgsToMap(0, 0, 1, 1, 1, 1, -1, -1, []byte(TestToken)))
}
