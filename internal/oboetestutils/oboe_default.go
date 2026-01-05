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
	"time"

	"github.com/solarwinds/apm-go/internal/oboe"
)

func GetDefaultSettingForTest() oboe.SettingsUpdateArgs {
	return oboe.SettingsUpdateArgs{
		Flags:                        "SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE",
		Value:                        1000000,
		Ttl:                          120 * time.Second,
		TriggerToken:                 []byte("token"),
		BucketCapacity:               1000000,
		BucketRate:                   1000000,
		MetricsFlushInterval:         -1,
		TriggerRelaxedBucketCapacity: 1000000,
		TriggerRelaxedBucketRate:     1000000,
		TriggerStrictBucketCapacity:  1000000,
		TriggerStrictBucketRate:      1000000,
	}
}
