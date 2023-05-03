// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReporterOptions(t *testing.T) {
	r := &ReporterOptions{}

	r.SetEventFlushInterval(20)
	assert.Equal(t, r.GetEventFlushInterval(), int64(20))

	r.SetMaxReqBytes(2000)
	assert.Equal(t, r.GetMaxReqBytes(), int64(2000))

	assert.Nil(t, r.validate())
}
