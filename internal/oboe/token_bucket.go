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

import (
	"github.com/solarwinds/apm-go/internal/metrics"
	"math"
	"strconv"
	"sync"
	"time"
)

type tokenBucket struct {
	ratePerSec float64
	capacity   float64
	available  float64
	last       time.Time
	lock       sync.Mutex
	metrics.RateCounts
}

func (b *tokenBucket) setRateCap(rate, cap float64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.ratePerSec = rate
	b.capacity = cap

	if b.available > b.capacity {
		b.available = b.capacity
	}
}

func (b *tokenBucket) count(sampled, hasMetadata, rateLimit bool) bool {
	b.RequestedInc()

	if !hasMetadata {
		b.SampledInc()
	}

	if !sampled {
		return sampled
	}

	if rateLimit {
		if ok := b.consume(1); !ok {
			b.LimitedInc()
			return false
		}
	}

	if hasMetadata {
		b.ThroughInc()
	}
	b.TracedInc()
	return sampled
}

func (b *tokenBucket) consume(size float64) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.update(time.Now())
	if b.available >= size {
		b.available -= size
		return true
	}
	return false
}

func (b *tokenBucket) update(now time.Time) {
	if b.available < b.capacity { // room for more tokens?
		delta := now.Sub(b.last) // calculate duration since last check
		b.last = now             // update time of last check
		if delta <= 0 {          // return if no delta or time went "backwards"
			return
		}
		newTokens := b.ratePerSec * delta.Seconds()               // # tokens generated since last check
		b.available = math.Min(b.capacity, b.available+newTokens) // add new tokens to bucket, but don't overfill
	}
}

func floatToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
