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
package solarwinds_apm

import (
	"testing"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type staticDecider struct {
	retval bool
}

func (d staticDecider) ShouldTraceRequestWithURL(layer string, traced bool, url string, ttMode reporter.TriggerTraceMode) (bool, string) {
	return d.retval, ""
}

// Note: I don't love these, but they work for this simple case.
var trueDecider decider = staticDecider{true}
var falseDecider decider = staticDecider{false}

func TestShouldSample(t *testing.T) {
	s := sampler{decider: trueDecider}
	params := sdktrace.SamplingParameters{}
	result := s.ShouldSample(params)
	assert.Equal(t, sdktrace.RecordAndSample, result.Decision)
	// todo: figure out how to test resulting tracestate
}

func TestShouldSampleFalse(t *testing.T) {
	s := sampler{decider: falseDecider}
	params := sdktrace.SamplingParameters{}
	result := s.ShouldSample(params)
	assert.Equal(t, sdktrace.Drop, result.Decision)
}

func TestNewSampler(t *testing.T) {
	s := NewSampler().(sampler)
	assert.Equal(t, s.decider, defaultDecider)
}

func TestDescription(t *testing.T) {
	s := NewSampler()
	assert.Equal(t, "SolarWinds APM Sampler", s.Description())
}
