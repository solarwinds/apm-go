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
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/solarwinds/apm-go/internal/utils"
)

type settingLambdaFromFile struct {
	Arguments *settingArguments `json:"arguments"`
	Flags     string            `json:"flags"`
	Timestamp int64             `json:"timestamp"`
	Ttl       int64             `json:"ttl"`
	Value     int64             `json:"value"`
}

type settingArguments struct {
	BucketCapacity               float64 `json:"BucketCapacity"`
	BucketRate                   float64 `json:"BucketRate"`
	MetricsFlushInterval         int     `json:"MetricsFlushInterval"`
	TriggerRelaxedBucketCapacity float64 `json:"TriggerRelaxedBucketCapacity"`
	TriggerRelaxedBucketRate     float64 `json:"TriggerRelaxedBucketRate"`
	TriggerStrictBucketCapacity  float64 `json:"TriggerStrictBucketCapacity"`
	TriggerStrictBucketRate      float64 `json:"TriggerStrictBucketRate"`
}

// N.B. this struct adheres to the types required by the oboe.Oboe interface. In the future,
// we should make the interface smarter about the incoming types so we're not converting from
// known types to []byte and back. A task has been created to track this work, though it also
// might make sense to do it when/if the GetSettings call is refactored.
type settingLambdaNormalized struct {
	flags []byte
	value int64
	ttl   int64
	args  map[string][]byte
}

// newSettingLambdaNormalized accepts settings in json-unmarshalled format
// for mapping to a format readable by oboe UpdateSetting.
func newSettingLambdaNormalized(fromFile *settingLambdaFromFile) *settingLambdaNormalized {
	flags := []byte(fromFile.Flags)

	var unusedToken = "TOKEN"
	args := utils.ArgsToMap(
		fromFile.Arguments.BucketCapacity,
		fromFile.Arguments.BucketRate,
		fromFile.Arguments.TriggerRelaxedBucketCapacity,
		fromFile.Arguments.TriggerRelaxedBucketRate,
		fromFile.Arguments.TriggerStrictBucketCapacity,
		fromFile.Arguments.TriggerStrictBucketRate,
		fromFile.Arguments.MetricsFlushInterval,
		-1,
		[]byte(unusedToken),
	)

	settingNorm := &settingLambdaNormalized{
		flags,
		fromFile.Value,
		fromFile.Ttl,
		args,
	}

	return settingNorm
}

// newSettingLambdaFromFile unmarshals sampling settings from a JSON file at a
// specific path in a specific format then returns values normalized for
// oboe UpdateSetting, else returns error.
func newSettingLambdaFromFile() (*settingLambdaNormalized, error) {
	settingFile, err := os.Open(settingsFileName)
	if err != nil {
		return nil, err
	}
	settingBytes, err := io.ReadAll(settingFile)
	if err != nil {
		return nil, err
	}
	// Settings file should be an array with a single settings object
	var settingLambdas []*settingLambdaFromFile
	if err = json.Unmarshal(settingBytes, &settingLambdas); err != nil {
		return nil, err
	}
	if len(settingLambdas) != 1 {
		return nil, errors.New("settings file is incorrectly formatted")
	}

	settingLambda := settingLambdas[0]

	return newSettingLambdaNormalized(settingLambda), nil
}
