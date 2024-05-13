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

package metrics

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTxnMap(t *testing.T) {
	m := newTxnMap(3)
	assert.EqualValues(t, 3, m.cap())
	assert.True(t, m.isWithinLimit("t1"))
	assert.True(t, m.isWithinLimit("t2"))
	assert.True(t, m.isWithinLimit("t3"))
	assert.False(t, m.isWithinLimit("t4"))
	assert.True(t, m.isWithinLimit("t2"))
	assert.True(t, m.isOverflowed())

	m.SetCap(4)
	m.reset()
	assert.EqualValues(t, 4, m.cap())
	assert.False(t, m.isOverflowed())
}
