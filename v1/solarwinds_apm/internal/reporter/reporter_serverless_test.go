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
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestServerless(t *testing.T) {
	serverless := os.Getenv("SW_APM_SERVERLESS")
	defer func() { os.Setenv("SW_APM_SERVERLESS", serverless) }()

	os.Setenv("SW_APM_SERVERLESS", "true")
	config.Load()

	var sb utils.SafeBuffer
	globalReporter = newServerlessReporter(&sb)
	r := globalReporter.(*serverlessReporter)

	ctx := newTestContext(t)
	ev1, err := ctx.newEvent(LabelInfo, "layer1")
	assert.NoError(t, err)
	assert.NoError(t, r.reportEvent(ctx, ev1))

	assert.Nil(t, r.Flush())
	arr := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	evtCnt := 0
	for _, s := range arr {
		sm := &ServerlessMessage{}
		assert.Nil(t, json.Unmarshal([]byte(s), sm))

		evtCnt += len(sm.Data.Events)

		for _, s := range sm.Data.Events {
			evtByte, err := base64.StdEncoding.DecodeString(s)
			assert.Nil(t, err)
			evt := string(evtByte)
			assert.Contains(t, evt, "layer1")
		}
	}
	assert.Equal(t, 1, evtCnt)
}

func TestServerlessShutdown(t *testing.T) {
	serverless := os.Getenv("SW_APM_SERVERLESS")
	defer func() { os.Setenv("SW_APM_SERVERLESS", serverless) }()

	os.Setenv("SW_APM_SERVERLESS", "true")
	config.Load()

	globalReporter = newServerlessReporter(os.Stderr)
	r := globalReporter.(*serverlessReporter)
	assert.Nil(t, r.ShutdownNow())
}
