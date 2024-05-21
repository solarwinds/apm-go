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
	stdlog "log"
	"os"
)

type settingLambda struct {
	Arguments *settingArguments `json:"arguments"`
	Flags     string            `json:"flags"`
	Layer     string            `json:"layer"`
	Timestamp int               `json:"timestamp"`
	Ttl       int64             `json:"ttl"`
	Stype     int               `json:"type"`
	Value     int               `json:"value"`
}

type settingArguments struct {
	BucketCapacity               float64 `json:"BucketCapacity"`
	BucketRate                   float64 `json:"BucketRate"`
	MetricsFlushInterval         int64   `json:"MetricsFlushInterval"`
	TriggerRelaxedBucketCapacity float64 `json:"TriggerRelaxedBucketCapacity"`
	TriggerRelaxedBucketRate     float64 `json:"TriggerRelaxedBucketRate"`
	TriggerStrictBucketCapacity  float64 `json:"TriggerStrictBucketCapacity"`
	TriggerStrictBucketRate      float64 `json:"TriggerStrictBucketRate"`
}

func newSettingLambdaFromFile() (*settingLambda, error) {
	// TODO tracing mode disabled if cannot open/parse/get all settings
	settingFile, err := os.Open("/tmp/solarwinds-apm-settings.json")
	if err != nil {
		return nil, err
	}
	settingBytes, err := io.ReadAll(settingFile)
	if err != nil {
		return nil, err
	}
	// Settings file should be an array with a single settings object
	var settingLambdas []settingLambda
	if err := json.Unmarshal(settingBytes, &settingLambdas); err != nil {
		return nil, err
	}
	if len(settingLambdas) != 1 {
		return nil, errors.New("settings file is incorrectly formatted")
	}

	var settingLambda settingLambda = settingLambdas[0]
	// tmp: debug
	stdlog.Printf("settingLambda: %v", settingLambda)
	stdlog.Printf("settingLambda.Arguments: %v", settingLambda.Arguments)

	return &settingLambda, nil
}
