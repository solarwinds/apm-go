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

package oboe

type SampleDecision struct {
	trace  bool
	rate   int
	source SampleSource
	// if the request is disabled from tracing in a per-transaction level or for
	// the entire service.
	enabled       bool
	xTraceOptsRsp string
	bucketCap     float64
	bucketRate    float64

	diceRolled bool
}

func (s SampleDecision) Trace() bool {
	return s.trace
}

func (s SampleDecision) XTraceOptsRsp() string {
	return s.xTraceOptsRsp
}

func (s SampleDecision) Enabled() bool {
	return s.enabled
}

func (s SampleDecision) BucketCapacity() float64 {
	return s.bucketCap
}

func (s SampleDecision) BucketCapacityStr() string {
	return floatToStr(s.BucketCapacity())
}

func (s SampleDecision) BucketRate() float64 {
	return s.bucketRate
}

func (s SampleDecision) BucketRateStr() string {
	return floatToStr(s.BucketRate())
}

func (s SampleDecision) SampleRate() int {
	return s.rate
}

func (s SampleDecision) SampleSource() SampleSource {
	return s.source
}
