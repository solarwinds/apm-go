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
package solarwinds_apm_test

import (
	"runtime/debug"
	"testing"
	"time"

	"context"

	solarwinds_apm "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm"
	g "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/graphtest"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
)

func TestSpans(t *testing.T) {
	r := reporter.SetTestReporter() // enable test reporter
	ctx := solarwinds_apm.NewContext(context.Background(), solarwinds_apm.NewTrace("myExample"))

	// make a cache request
	l := solarwinds_apm.BeginCacheSpan(ctx, "redis", "INCR", "key31", "redis.net", true)
	// ... client.Incr(key) ...
	time.Sleep(20 * time.Millisecond)
	l.Error("CacheTimeoutError", "Cache request timeout error!")
	l.End()

	// make an RPC request (no trace propagation in this example)
	l = solarwinds_apm.BeginRPCSpan(ctx, "myServiceClient", "thrift", "incrKey", "service.net")
	// ... service.incrKey(key) ...
	time.Sleep(time.Millisecond)
	l.End()

	// make a query span
	l = solarwinds_apm.BeginQuerySpan(ctx, "querySpan", "SELECT * FROM TEST_TABLE",
		"MySQL", "remote.host", solarwinds_apm.KeyBackTrace, string(debug.Stack()))
	time.Sleep(time.Millisecond)
	l.End()

	solarwinds_apm.End(ctx)

	r.Close(9)
	g.AssertGraph(t, r.EventBufs, 9, g.AssertNodeMap{
		// entry event should have no edges
		{"myExample", "entry"}: {},
		{"redis", "entry"}: {Edges: g.Edges{{"myExample", "entry"}}, Callback: func(n g.Node) {
			assert.Equal(t, "redis.net", n.Map["RemoteHost"])
			assert.Equal(t, "INCR", n.Map["KVOp"])
			assert.Equal(t, "key31", n.Map["KVKey"])
			assert.Equal(t, true, n.Map["KVHit"])
		}},
		{"redis", "error"}: {Edges: g.Edges{{"redis", "entry"}}, Callback: func(n g.Node) {
			assert.Equal(t, "CacheTimeoutError", n.Map["ErrorClass"])
			assert.Equal(t, "Cache request timeout error!", n.Map["ErrorMsg"])
		}},
		{"redis", "exit"}: {Edges: g.Edges{{"redis", "error"}}},
		{"myServiceClient", "entry"}: {Edges: g.Edges{{"myExample", "entry"}}, Callback: func(n g.Node) {
			assert.Equal(t, "service.net", n.Map["RemoteHost"])
			assert.Equal(t, "incrKey", n.Map["RemoteController"])
			assert.Equal(t, "thrift", n.Map["RemoteProtocol"])
			assert.Equal(t, "rsc", n.Map["Spec"])
		}},
		{"myServiceClient", "exit"}: {Edges: g.Edges{{"myServiceClient", "entry"}}},
		{"querySpan", "entry"}: {Edges: g.Edges{{"myExample", "entry"}}, Callback: func(n g.Node) {
			assert.Equal(t, "query", n.Map["Spec"])
			assert.Equal(t, "remote.host", n.Map["RemoteHost"])
			assert.Equal(t, "SELECT * FROM TEST_TABLE", n.Map["Query"])
			assert.Equal(t, "MySQL", n.Map["Flavor"])
			assert.NotNil(t, n.Map[solarwinds_apm.KeyBackTrace])
		}},
		{"querySpan", "exit"}: {Edges: g.Edges{{"querySpan", "entry"}}},
		{"myExample", "exit"}: {Edges: g.Edges{{"redis", "exit"}, {"myServiceClient", "exit"}, {"querySpan", "exit"}, {"myExample", "entry"}}},
	})
}
