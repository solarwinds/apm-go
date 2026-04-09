// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
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

package reporter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMetricsPublisher(t *testing.T) {
	p := NewMetricsPublisher()

	require.NotNil(t, p)
}

func TestMetricsPublisherGetMetricsRegistryBeforeStart(t *testing.T) {
	p := NewMetricsPublisher()

	require.Nil(t, p.GetMetricsRegistry())
}

func TestMetricsPublisherShutdownWhenNotConfigured(t *testing.T) {
	p := NewMetricsPublisher()

	require.NoError(t, p.Shutdown())
}
