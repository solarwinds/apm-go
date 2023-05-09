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
//go:build basictracer
// +build basictracer

//
// Behind a build tag avoid adding a dependency on basictracer-go

package opentracing

import (
	"testing"

	bt "github.com/opentracing/basictracer-go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/harness"
	mt "github.com/solarwindscloud/solarwinds-apm-go/v1/contrib/multitracer"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
)

// This test sets up SolarWinds Observability Tracer and the OT "BasicTracer" side by side
func TestMultiTracerAPMBasicTracerAPICheck(t *testing.T) {
	_ = reporter.SetTestReporter(reporter.TestReporterDisableDefaultSetting(true)) // set up test reporter
	multiTracer := &mt.MultiTracer{
		Tracers: []opentracing.Tracer{
			NewTracer(),
			bt.NewWithOptions(bt.Options{
				Recorder:     bt.NewInMemoryRecorder(),
				ShouldSample: func(traceID uint64) bool { return true }, // always sample
			}),
		}}

	harness.RunAPIChecks(t, func() (tracer opentracing.Tracer, closer func()) {
		return multiTracer, nil
	},
		harness.CheckBaggageValues(false),
		harness.CheckInject(true),
		harness.CheckExtract(true),
		harness.UseProbe(&multiApiCheckProbe{
			mt:     multiTracer,
			probes: []harness.APICheckProbe{apiCheckProbe{}, nil},
		}),
	)
}

// This test sets up the OT "BasicTracer" wrapped in a MultiTracer
func TestMultiTracerBasicTracerAPICheck(t *testing.T) {
	_ = reporter.SetTestReporter(reporter.TestReporterDisableDefaultSetting(true)) // set up test reporter
	harness.RunAPIChecks(t, func() (tracer opentracing.Tracer, closer func()) {
		return &mt.MultiTracer{
			Tracers: []opentracing.Tracer{
				bt.NewWithOptions(bt.Options{
					Recorder:     bt.NewInMemoryRecorder(),
					ShouldSample: func(traceID uint64) bool { return true }, // always sample
				}),
			}}, nil
	},
		harness.CheckBaggageValues(false),
		harness.CheckInject(true),
		harness.CheckExtract(true),
	)
}
