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

	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingLambdaNormalized(t *testing.T) {
	settingArgs := settingArguments{
		1,
		1,
		1,
		1,
		1,
		1,
		1,
	}
	fromFile := settingLambdaFromFile{
		&settingArgs,
		"SAMPLE_START,SAMPLE_THROUGH_ALWAYS,SAMPLE_BUCKET_ENABLED,TRIGGER_TRACE",
		1715900164,
		120,
		1000000,
	}
	result := newSettingLambdaNormalized(&fromFile)

	assert.Equal(
		t,
		[]byte{0x53, 0x41, 0x4d, 0x50, 0x4c, 0x45, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x54, 0x2c, 0x53, 0x41, 0x4d, 0x50, 0x4c, 0x45, 0x5f, 0x54, 0x48, 0x52, 0x4f, 0x55, 0x47, 0x48, 0x5f, 0x41, 0x4c, 0x57, 0x41, 0x59, 0x53, 0x2c, 0x53, 0x41, 0x4d, 0x50, 0x4c, 0x45, 0x5f, 0x42, 0x55, 0x43, 0x4b, 0x45, 0x54, 0x5f, 0x45, 0x4e, 0x41, 0x42, 0x4c, 0x45, 0x44, 0x2c, 0x54, 0x52, 0x49, 0x47, 0x47, 0x45, 0x52, 0x5f, 0x54, 0x52, 0x41, 0x43, 0x45},
		result.flags,
	)
	assert.Equal(t, result.value, int64(1000000))
	assert.Equal(t, result.ttl, int64(120))
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceRelaxedBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceRelaxedBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceStrictBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceStrictBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x1, 0x0, 0x0, 0x0},
		result.args[constants.KvMetricsFlushInterval],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result.args[constants.KvMaxTransactions],
	)
	assert.Equal(
		t,
		[]byte{0x54, 0x4f, 0x4b, 0x45, 0x4e},
		result.args[constants.KvSignatureKey],
	)
}

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
		[]byte{0x53, 0x41, 0x4d, 0x50, 0x4c, 0x45, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x54, 0x2c, 0x53, 0x41, 0x4d, 0x50, 0x4c, 0x45, 0x5f, 0x54, 0x48, 0x52, 0x4f, 0x55, 0x47, 0x48, 0x5f, 0x41, 0x4c, 0x57, 0x41, 0x59, 0x53, 0x2c, 0x53, 0x41, 0x4d, 0x50, 0x4c, 0x45, 0x5f, 0x42, 0x55, 0x43, 0x4b, 0x45, 0x54, 0x5f, 0x45, 0x4e, 0x41, 0x42, 0x4c, 0x45, 0x44, 0x2c, 0x54, 0x52, 0x49, 0x47, 0x47, 0x45, 0x52, 0x5f, 0x54, 0x52, 0x41, 0x43, 0x45},
		result.flags,
	)
	assert.Equal(t, result.value, int64(1000000))
	assert.Equal(t, result.ttl, int64(120))
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceRelaxedBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceRelaxedBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceStrictBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result.args[constants.KvTriggerTraceStrictBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x1, 0x0, 0x0, 0x0},
		result.args[constants.KvMetricsFlushInterval],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result.args[constants.KvMaxTransactions],
	)
	assert.Equal(
		t,
		[]byte{0x54, 0x4f, 0x4b, 0x45, 0x4e},
		result.args[constants.KvSignatureKey],
	)

	require.NoError(t, os.Remove(settingsFileName))
}
