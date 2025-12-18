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

package oboe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsUpdater_QueriesAndUpdatesSettings(t *testing.T) {
	// Setup mock HTTP server with specific settings
	token := "abcdaaaaaakaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	serviceName := "test-service"
	expectedSettings := httpSettings{
		Flags:     "SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE",
		Value:     500000,
		Ttl:       90,
		Timestamp: time.Now().Unix(),
		Arguments: &httpSettingArguments{
			BucketCapacity:               750000,
			BucketRate:                   850000,
			MetricsFlushInterval:         45,
			TriggerRelaxedBucketCapacity: 200,
			TriggerRelaxedBucketRate:     250,
			TriggerStrictBucketCapacity:  20,
			TriggerStrictBucketRate:      25,
		},
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, "/v1/settings/"+serviceName+"/unknown", r.URL.Path)
		assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(expectedSettings)
		require.NoError(t, err)
	}))
	defer server.Close()

	config.Load(
		config.WithServiceKey(token+":"+serviceName),
		func(c *config.Config) {
			c.SettingsURL = server.URL
		},
		func(c *config.Config) {
			c.Enabled = true
		},
	)

	o := NewOboe()
	updater, err := NewSettingsUpdater(o)
	require.NoError(t, err)
	require.NotNil(t, updater)

	assert.False(t, o.HasDefaultSetting(), "oboe should not have default settings initially")

	stop := updater.Start(t.Context())
	defer stop()

	// Wait for the HTTP server to be called and settings to be synchronized
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	synchronized := false
	for !synchronized {
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for synchronization. requestCount=%d", requestCount)
		case <-ticker.C:
			if requestCount > 0 {
				synchronized = true
			}
		}
	}

	assert.Equal(t, 1, requestCount, "HTTP server should have been called once")

	storedSettings := o.GetSetting()
	require.NotNil(t, storedSettings, "oboe should have stored settings")
	assert.True(t, o.HasDefaultSetting(), "oboe should indicate it has default settings")

	assert.Equal(t, int(expectedSettings.Value), storedSettings.value, "sample rate should match")
	assert.Equal(t, time.Duration(expectedSettings.Ttl)*time.Second, storedSettings.ttl, "TTL should match")
	assert.Equal(t, float64(expectedSettings.Arguments.BucketCapacity), storedSettings.bucket.capacity, "bucket capacity should match")
	assert.Equal(t, float64(expectedSettings.Arguments.BucketRate), storedSettings.bucket.ratePerSec, "bucket rate should match")
	assert.Equal(t, float64(expectedSettings.Arguments.TriggerRelaxedBucketCapacity), storedSettings.triggerTraceRelaxedBucket.capacity, "trigger relaxed bucket capacity should match")
	assert.Equal(t, float64(expectedSettings.Arguments.TriggerRelaxedBucketRate), storedSettings.triggerTraceRelaxedBucket.ratePerSec, "trigger relaxed bucket rate should match")
	assert.Equal(t, float64(expectedSettings.Arguments.TriggerStrictBucketCapacity), storedSettings.triggerTraceStrictBucket.capacity, "trigger strict bucket capacity should match")
	assert.Equal(t, float64(expectedSettings.Arguments.TriggerStrictBucketRate), storedSettings.triggerTraceStrictBucket.ratePerSec, "trigger strict bucket rate should match")
}
