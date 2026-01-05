// © 2024 SolarWinds Worldwide, LLC. All rights reserved.
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

//go:build !windows

package oboe

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingLambdaFromFileErrorOpen(t *testing.T) {
	require.NoFileExists(t, settingsFileName)
	res, err := newSettingLambdaFromFile()
	assert.Nil(t, res)
	assert.Error(t, err)
}

func TestNewSettingLambdaFromFileErrorUnmarshal(t *testing.T) {
	require.NoFileExists(t, settingsFileName)

	content := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(settingsFileName, content, 0644))
	res, err := newSettingLambdaFromFile()
	assert.Nil(t, res)
	assert.Error(t, err)

	require.NoError(t, os.Remove(settingsFileName))
}

func TestNewSettingLambdaFromFileErrorLen(t *testing.T) {
	require.NoFileExists(t, settingsFileName)

	content := []byte("[]")
	require.NoError(t, os.WriteFile(settingsFileName, content, 0644))
	res, err := newSettingLambdaFromFile()
	assert.Nil(t, res)
	assert.Error(t, err)

	require.NoError(t, os.Remove(settingsFileName))
}

func TestNewSettingLambdaFromFile(t *testing.T) {
	require.NoFileExists(t, settingsFileName)

	content := []byte("[{\"arguments\":{\"BucketCapacity\":1,\"BucketRate\":1,\"MetricsFlushInterval\":1,\"TriggerRelaxedBucketCapacity\":1,\"TriggerRelaxedBucketRate\":1,\"TriggerStrictBucketCapacity\":1,\"TriggerStrictBucketRate\":1},\"flags\":\"SAMPLE_START,SAMPLE_THROUGH_ALWAYS,SAMPLE_BUCKET_ENABLED,TRIGGER_TRACE\",\"layer\":\"\",\"timestamp\":1715900164,\"ttl\":120,\"type\":0,\"value\":1000000}]")
	require.NoError(t, os.WriteFile(settingsFileName, content, 0644))
	result, err := newSettingLambdaFromFile()
	assert.Nil(t, err)
	assert.Equal(
		t,
		"SAMPLE_START,SAMPLE_THROUGH_ALWAYS,SAMPLE_BUCKET_ENABLED,TRIGGER_TRACE",
		result.Flags,
	)
	assert.Equal(t, result.Value, int64(1000000))
	assert.Equal(t, result.Ttl, int64(120))
	assert.Equal(
		t,
		float64(1),
		result.Arguments.BucketCapacity,
	)
	assert.Equal(
		t,
		float64(1),
		result.Arguments.BucketRate,
	)
	assert.Equal(
		t,
		float64(1),
		result.Arguments.TriggerRelaxedBucketCapacity,
	)
	assert.Equal(
		t,
		float64(1),
		result.Arguments.TriggerRelaxedBucketRate,
	)
	assert.Equal(
		t,
		float64(1),
		result.Arguments.TriggerStrictBucketCapacity,
	)
	assert.Equal(
		t,
		float64(1),
		result.Arguments.TriggerStrictBucketRate,
	)
	assert.Equal(
		t,
		int(1),
		result.Arguments.MetricsFlushInterval,
	)
	require.NoError(t, os.Remove(settingsFileName))
}
