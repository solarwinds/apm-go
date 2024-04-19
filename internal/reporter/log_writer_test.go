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
package reporter

import (
	"github.com/solarwinds/apm-go/internal/utils"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogWriter(t *testing.T) {
	sb := &utils.SafeBuffer{}
	eventWriter := newLogWriter(false, sb, 1e6)
	_, err := eventWriter.Write(EventWT, []byte("hello event"))
	require.NoError(t, err)
	assert.Equal(t, 0, sb.Len())
	require.NoError(t, eventWriter.Flush())
	assert.Equal(t, "{\"apm-data\":{\"events\":[\"aGVsbG8gZXZlbnQ=\"]}}\n", sb.String())

	sb.Reset()
	metricWriter := newLogWriter(true, sb, 1e6)
	_, err = metricWriter.Write(MetricWT, []byte("hello metric"))
	require.NoError(t, err)
	assert.Equal(t, "{\"apm-data\":{\"metrics\":[\"aGVsbG8gbWV0cmlj\"]}}\n", sb.String())
	assert.NotNil(t, metricWriter.Flush())

	sb.Reset()
	writer := newLogWriter(false, sb, 15)
	n, err := writer.Write(EventWT, []byte("hello event"))
	assert.Zero(t, n)
	assert.Error(t, err)

	_, err = writer.Write(EventWT, []byte("hello"))
	require.NoError(t, err)
	assert.Zero(t, sb.Len())
	_, err = writer.Write(EventWT, []byte(" event"))
	require.NoError(t, err)
	assert.Equal(t, 37, sb.Len())
	assert.Equal(t, "{\"apm-data\":{\"events\":[\"aGVsbG8=\"]}}\n", sb.String())
	require.NoError(t, writer.Flush())
	assert.Equal(t, "{\"apm-data\":{\"events\":[\"aGVsbG8=\"]}}\n{\"apm-data\":{\"events\":[\"IGV2ZW50\"]}}\n",
		sb.String())

}
