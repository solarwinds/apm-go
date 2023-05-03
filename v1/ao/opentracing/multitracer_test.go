package opentracing

import (
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/harness"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/ao/internal/reporter"
	mt "github.com/solarwindscloud/solarwinds-apm-go/v1/contrib/multitracer"
)

// This test sets up the SolarWinds Observability Tracer wrapped in a MultiTracer
func TestMultiTracerAPICheck(t *testing.T) {
	_ = reporter.SetTestReporter(reporter.TestReporterDisableDefaultSetting(true)) // set up test reporter
	multiTracer := &mt.MultiTracer{Tracers: []opentracing.Tracer{NewTracer()}}

	harness.RunAPIChecks(t, func() (tracer opentracing.Tracer, closer func()) {
		return multiTracer, nil
	},
		harness.CheckBaggageValues(false),
		harness.CheckInject(true),
		harness.CheckExtract(true),
		harness.UseProbe(&multiApiCheckProbe{
			mt:     multiTracer,
			probes: []harness.APICheckProbe{apiCheckProbe{}},
		}),
	)
}

type multiApiCheckProbe struct {
	mt     *mt.MultiTracer
	probes []harness.APICheckProbe
}

func (m *multiApiCheckProbe) SameTrace(first, second opentracing.Span) bool {
	sp1 := first.(*mt.MultiSpan)
	sp2 := second.(*mt.MultiSpan)

	for i := range m.mt.Tracers {
		if m.probes[i] == nil {
			continue
		}
		if !m.probes[i].SameTrace(sp1.Spans[i], sp2.Spans[i]) {
			return false
		}
	}
	return true
}

func (m *multiApiCheckProbe) SameSpanContext(span opentracing.Span, spanCtx opentracing.SpanContext) bool {
	sp := span.(*mt.MultiSpan)
	sc := spanCtx.(*mt.MultiSpanContext)

	for i := range m.mt.Tracers {
		if m.probes[i] == nil {
			continue
		}
		if !m.probes[i].SameSpanContext(sp.Spans[i], sc.SpanContexts[i]) {
			return false
		}
	}
	return true
}
