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

//lint:file-ignore SA1012 We're testing that our own code does not panic

package solarwinds_apm

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	g "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/graphtest"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	r := reporter.SetTestReporter()

	ctx := context.Background()
	assert.Empty(t, MetadataString(ctx))
	tr := NewTrace("test").(*apmTrace)

	xt := tr.apmCtx.MetadataString()
	//	assert.True(t, IsSampled(ctx), "%T", tr.apmCtx)

	var traceKey = struct{}{}

	ctx2 := context.WithValue(ctx, traceKey, tr)
	assert.Equal(t, ctx2.Value(traceKey), tr)
	assert.Equal(t, ctx2.Value(traceKey).(*apmTrace).apmCtx.MetadataString(), xt)

	ctxx := tr.apmCtx.Copy()
	lbl := &spanLabeler{"L1"}
	tr2 := &apmTrace{layerSpan: layerSpan{span: span{apmCtx: ctxx, labeler: lbl}}}
	ctx3 := context.WithValue(ctx2, traceKey, tr2)
	assert.Equal(t, ctx3.Value(traceKey), tr2)

	ctxx2 := tr2.apmCtx.Copy()
	tr3 := &apmTrace{layerSpan: layerSpan{span: span{apmCtx: ctxx2}}}
	ctx4 := context.WithValue(ctx3, traceKey, tr3)
	assert.Equal(t, ctx4.Value(traceKey), tr3)

	r.Close(1)
	g.AssertGraph(t, r.EventBufs, 1, g.AssertNodeMap{{"test", "entry"}: {}})
}

func TestTraceFromContext(t *testing.T) {
	r := reporter.SetTestReporter()
	tr := NewTrace("TestTFC")
	ctx := NewContext(context.Background(), tr)
	trFC := TraceFromContext(ctx)
	assert.Equal(t, tr.ExitMetadata(), trFC.ExitMetadata())
	assert.Len(t, tr.ExitMetadata(), 60)

	trN := TraceFromContext(context.Background()) // no trace bound to this ctx
	assert.Len(t, trN.ExitMetadata(), 0)

	r.Close(1)
	g.AssertGraph(t, r.EventBufs, 1, g.AssertNodeMap{{"TestTFC", "entry"}: {}})
}

func TestContextIsSampled(t *testing.T) {
	// no context: not sampled
	assert.False(t, IsSampled(context.Background()))
	// sampled context
	_ = reporter.SetTestReporter()
	tr := NewTrace("TestTFC")
	ctx := NewContext(context.Background(), tr)
	assert.True(t, IsSampled(ctx))
}

func TestNullSpan(t *testing.T) {
	// enable reporting to test reporter
	r := reporter.SetTestReporter()

	ctx := NewContext(context.Background(), NewTrace("TestNullSpan")) // reports event
	l1, ctxL := BeginSpan(ctx, "L1")                                  // reports event
	assert.True(t, l1.IsReporting())
	assert.Equal(t, l1.MetadataString(), MetadataString(ctxL))
	assert.Len(t, l1.MetadataString(), 60)

	l1.End() // reports event
	assert.False(t, l1.IsReporting())
	assert.Empty(t, l1.MetadataString())

	p1 := l1.BeginProfile("P2") // try to start profile after end: no effect
	p1.End()

	c1 := l1.BeginSpan("C1") // child after parent ended
	assert.IsType(t, c1, nullSpan{})
	assert.False(t, c1.IsReporting())
	assert.False(t, c1.IsSampled())
	assert.False(t, c1.ok())
	assert.Empty(t, c1.MetadataString())
	c1.addChildEdge(l1.apmContext())
	c1.addProfile(p1)

	nctx := c1.apmContext()
	assert.Equal(t, reflect.TypeOf(nctx).Elem().Name(), "nullContext")
	assert.IsType(t, reflect.TypeOf(nctx.Copy()).Elem().Name(), "nullContext")

	r.Close(3)
	g.AssertGraph(t, r.EventBufs, 3, g.AssertNodeMap{
		{"TestNullSpan", "entry"}: {},
		{"L1", "entry"}:           {Edges: g.Edges{{"TestNullSpan", "entry"}}},
		{"L1", "exit"}:            {Edges: g.Edges{{"L1", "entry"}}},
	})
}

func TestNilContext(t *testing.T) {
	assert.NotPanics(t, func() { assert.IsType(t, &nullTrace{}, TraceFromContext(nil)) })
	assert.NotPanics(t, func() { assert.IsType(t, nullSpan{}, FromContext(nil)) })
	assert.NotPanics(t, func() { assert.NoError(t, SetTransactionName(nil, "tx")) })
	assert.NotPanics(t, func() { Error(nil, "error", "msg") })
	assert.NotPanics(t, func() { End(nil) })
	assert.NotPanics(t, func() { EndTrace(nil) })
	assert.NotPanics(t, func() { Info(nil, "k", "v") })
	assert.NotPanics(t, func() { assert.Empty(t, MetadataString(nil)) })
	assert.NotPanics(t, func() { assert.False(t, IsSampled(nil)) })

	baselineV, _ := version.NewSemver("1.15")
	v, _ := version.NewSemver(strings.TrimLeft(runtime.Version(), "go"))

	if v.GreaterThanOrEqual(baselineV) {
		// Go 1.15 introduced panic behavior when passing nil contexts around https://golang.org/doc/go1.15#context
		assert.Panics(t, func() { NewContext(nil, TraceFromContext(nil)) })
	} else {
		assert.NotPanics(t, func() { NewContext(nil, TraceFromContext(nil)) })
	}
}

func BenchmarkTraceFromContextEmpty(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TraceFromContext(ctx)
	}
}