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

package metrics

import "sync/atomic"

// RateCounts is the rate counts reported by trace sampler
type RateCounts struct{ requested, sampled, limited, traced, through, ttraced int64 }

var rateCountsAggregator = &RateCounts{}

func RatesAggregator() *RateCounts {
	return rateCountsAggregator
}

// FlushRateCounts reset the counters and returns the current value
func (c *RateCounts) FlushRateCounts() *RateCounts {
	return &RateCounts{
		requested: atomic.SwapInt64(&c.requested, 0),
		sampled:   atomic.SwapInt64(&c.sampled, 0),
		limited:   atomic.SwapInt64(&c.limited, 0),
		traced:    atomic.SwapInt64(&c.traced, 0),
		through:   atomic.SwapInt64(&c.through, 0),
		ttraced:   atomic.SwapInt64(&c.ttraced, 0),
	}
}

func (c *RateCounts) RequestedInc() {
	atomic.AddInt64(&c.requested, 1)
}

func (c *RateCounts) Requested() int64 {
	return atomic.LoadInt64(&c.requested)
}

func (c *RateCounts) SampledInc() {
	atomic.AddInt64(&c.sampled, 1)
}

func (c *RateCounts) Sampled() int64 {
	return atomic.LoadInt64(&c.sampled)
}

func (c *RateCounts) LimitedInc() {
	atomic.AddInt64(&c.limited, 1)
}

func (c *RateCounts) Limited() int64 {
	return atomic.LoadInt64(&c.limited)
}

func (c *RateCounts) TracedInc() {
	atomic.AddInt64(&c.traced, 1)
}

func (c *RateCounts) Traced() int64 {
	return atomic.LoadInt64(&c.traced)
}

func (c *RateCounts) ThroughInc() {
	atomic.AddInt64(&c.through, 1)
}

func (c *RateCounts) Through() int64 {
	return atomic.LoadInt64(&c.through)
}

func (c *RateCounts) TriggerTraceInc() {
	atomic.AddInt64(&c.ttraced, 1)
}

func (c *RateCounts) TriggerTrace() int64 {
	return atomic.LoadInt64(&c.ttraced)
}
