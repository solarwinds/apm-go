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
package opentracing

import (
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/harness"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
)

// apiCheckProbe exposes methods for testing data recorded by a Tracer.
type apiCheckProbe struct{}

// SameTrace helps tests assert that this tracer's spans are from the same trace.
func (apiCheckProbe) SameTrace(first, second opentracing.Span) bool {
	sp1 := first.(*spanImpl)
	sp2 := second.(*spanImpl)
	return sp1.context.trace.LoggableTraceID() == sp2.context.trace.LoggableTraceID()
}

// SameSpanContext helps tests assert that a span and a context are from the same trace and span.
func (apiCheckProbe) SameSpanContext(span opentracing.Span, spanCtx opentracing.SpanContext) bool {
	sp := span.(*spanImpl)
	sc := spanCtx.(spanContext)
	var md1, md2 string
	md1 = sp.context.span.MetadataString()
	if sc.span == nil {
		md1 = sc.remoteMD
	} else {
		md1 = sc.span.MetadataString()
	}
	md2 = sp.context.span.MetadataString()
	return md1 == md2
}

func TestAPICheck(t *testing.T) {
	_ = reporter.SetTestReporter(reporter.TestReporterDisableDefaultSetting(true)) // set up test reporter
	harness.RunAPIChecks(t, func() (tracer opentracing.Tracer, closer func()) {
		return NewTracer(), nil
	},
		harness.CheckBaggageValues(true),
		harness.CheckInject(true),
		harness.CheckExtract(true),
		harness.UseProbe(apiCheckProbe{}),
	)
}
