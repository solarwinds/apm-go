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

package utils

import (
	"testing"

	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/stretchr/testify/assert"
)

const TestToken = "TOKEN"

func TestArgsToMapAllSet(t *testing.T) {
	result := ArgsToMap(1, 1, 1, 1, 1, 1, 1, 1, []byte(TestToken))

	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result[constants.KvBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result[constants.KvBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result[constants.KvTriggerTraceRelaxedBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result[constants.KvTriggerTraceRelaxedBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result[constants.KvTriggerTraceStrictBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf0, 0x3f},
		result[constants.KvTriggerTraceStrictBucketRate],
	)
	assert.Equal(
		t,
		[]byte{0x1, 0x0, 0x0, 0x0},
		result[constants.KvMetricsFlushInterval],
	)
	assert.Equal(
		t,
		[]byte{0x1, 0x0, 0x0, 0x0},
		result[constants.KvMaxTransactions],
	)
	assert.Equal(
		t,
		[]byte{0x54, 0x4f, 0x4b, 0x45, 0x4e},
		result[constants.KvSignatureKey],
	)
}

func TestArgsToMapAllUnset(t *testing.T) {
	result := ArgsToMap(-1, -1, -1, -1, -1, -1, -1, -1, []byte(TestToken))

	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvBucketRate],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvTriggerTraceRelaxedBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvTriggerTraceRelaxedBucketRate],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvTriggerTraceStrictBucketCapacity],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvTriggerTraceStrictBucketRate],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvMetricsFlushInterval],
	)
	assert.Equal(
		t,
		[]byte(nil),
		result[constants.KvMaxTransactions],
	)
	assert.Equal(
		t,
		[]byte{0x54, 0x4f, 0x4b, 0x45, 0x4e},
		result[constants.KvSignatureKey],
	)
}
