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
	"github.com/solarwindscloud/solarwinds-apm-go/internal/config"
	"testing"

	"github.com/coocood/freecache"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	cache := &urlCache{freecache.NewCache(1024 * 1024)}

	cache.setURLTrace("traced_1", TraceEnabled)
	cache.setURLTrace("not_traced_1", TraceDisabled)
	assert.Equal(t, int64(2), cache.EntryCount())

	trace, err := cache.getURLTrace("traced_1")
	assert.Nil(t, err)
	assert.Equal(t, TraceEnabled, trace)
	assert.Equal(t, int64(1), cache.HitCount())

	trace, err = cache.getURLTrace("not_traced_1")
	assert.Nil(t, err)
	assert.Equal(t, TraceDisabled, trace)
	assert.Equal(t, int64(2), cache.HitCount())

	trace, err = cache.getURLTrace("non_exist_1")
	assert.NotNil(t, err)
	assert.Equal(t, TraceUnknown, trace)
	assert.Equal(t, int64(2), cache.HitCount())
	assert.Equal(t, int64(1), cache.MissCount())
}

func TestUrlFilter(t *testing.T) {
	filter := newURLFilters()
	filter.loadConfig([]config.TransactionFilter{
		{Type: "url", RegEx: `user\d{3}`, Tracing: config.DisabledTracingMode},
		{Type: "url", Extensions: []string{"png", "jpg"}, Tracing: config.DisabledTracingMode},
	})

	assert.Equal(t, TraceDisabled, filter.getTracingMode("user123"))
	assert.Equal(t, int64(1), filter.cache.EntryCount())
	assert.Equal(t, int64(0), filter.cache.HitCount())

	assert.Equal(t, TraceUnknown, filter.getTracingMode("test123"))
	assert.Equal(t, int64(2), filter.cache.EntryCount())
	assert.Equal(t, int64(2), filter.cache.MissCount())

	assert.Equal(t, TraceDisabled, filter.getTracingMode("user200"))
	assert.Equal(t, int64(3), filter.cache.EntryCount())
	assert.Equal(t, int64(0), filter.cache.HitCount())

	assert.Equal(t, TraceDisabled, filter.getTracingMode("user123"))
	assert.Equal(t, int64(3), filter.cache.EntryCount())
	assert.Equal(t, int64(1), filter.cache.HitCount())

	assert.Equal(t, TraceDisabled, filter.getTracingMode("http://user.com/eric/avatar.png"))
	assert.Equal(t, int64(4), filter.cache.EntryCount())
}
