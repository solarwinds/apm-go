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
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrappers(t *testing.T) {
	require.NoError(t, os.Unsetenv(envSolarWindsAPMCollector))
	require.NoError(t, os.Unsetenv(envSolarWindsAPMHistogramPrecision))
	Load()

	assert.NotEqual(t, nil, conf)
	assert.Equal(t, getFieldDefaultValue(&Config{}, "Collector"), GetCollector())
	assert.Equal(t, ToInteger(getFieldDefaultValue(&Config{}, "Precision")), GetPrecision())

	assert.NotEqual(t, nil, ReporterOpts())
}
