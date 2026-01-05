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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServerHandler is a helper function to create a test server with a custom handler
func testServerHandler(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// testServerWithResponse is a helper function to create a test server that returns a JSON response
func testServerWithResponse(t *testing.T, statusCode int, response interface{}) *httptest.Server {
	return testServerHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if response != nil {
			err := json.NewEncoder(w).Encode(response)
			require.NoError(t, err)
		}
	})
}

func TestBuildURL(t *testing.T) {
	svc := newSettingsService("https://api.example.com", "my-service", "my-host", "token")
	expected := "https://api.example.com/v1/settings/my-service/my-host"

	actual := svc.buildURL()

	assert.Equal(t, expected, actual)
}

func TestSetAuthHeaders(t *testing.T) {
	svc := newSettingsService("https://api.example.com", "service", "host", "my-bearer-token")
	req, err := http.NewRequest(http.MethodGet, "https://api.example.com/test", nil)
	require.NoError(t, err)

	svc.setAuthHeaders(req)

	assert.Equal(t, "Bearer my-bearer-token", req.Header.Get("Authorization"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
}

func TestGetSettings_Success(t *testing.T) {
	mockResponse := httpSettings{
		Flags:     "SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE",
		Value:     1000000,
		Ttl:       120,
		Timestamp: time.Now().Unix(),
		Arguments: &httpSettingArguments{
			BucketCapacity:               1000000,
			BucketRate:                   1000000,
			MetricsFlushInterval:         30,
			TriggerRelaxedBucketCapacity: 100,
			TriggerRelaxedBucketRate:     100,
			TriggerStrictBucketCapacity:  10,
			TriggerStrictBucketRate:      10,
		},
	}

	server := testServerHandler(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/settings/test-service/test-host", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(mockResponse)
		require.NoError(t, err)
	})
	defer server.Close()

	svc := newSettingsService(server.URL, "test-service", "test-host", "test-token")
	settings, err := svc.getSettings(context.Background())

	require.NoError(t, err)
	require.NotNil(t, settings)
	assert.Equal(t, mockResponse.Flags, settings.Flags)
	assert.Equal(t, mockResponse.Value, settings.Value)
	assert.Equal(t, mockResponse.Ttl, settings.Ttl)
	assert.Equal(t, mockResponse.Arguments.BucketCapacity, settings.Arguments.BucketCapacity)
}

func TestGetSettings_Unauthorized(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"401 Unauthorized", http.StatusUnauthorized},
		{"403 Forbidden", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServerHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})
			defer server.Close()

			svc := newSettingsService(server.URL, "test-service", "test-host", "invalid-token")
			settings, err := svc.getSettings(context.Background())

			assert.Nil(t, settings)
			require.Error(t, err)
			assert.ErrorIs(t, err, config.ErrInvalidServiceKey)
		})
	}
}

func TestGetSettings_UnexpectedStatusCode(t *testing.T) {
	server := testServerHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	})
	defer server.Close()

	svc := newSettingsService(server.URL, "test-service", "test-host", "test-token")
	settings, err := svc.getSettings(context.Background())

	assert.Nil(t, settings)
	require.Error(t, err)
	assert.NotErrorIs(t, err, config.ErrInvalidServiceKey, "should not be auth error")
	assert.Contains(t, err.Error(), "unexpected status code 500", "error should mention status code")
	assert.Contains(t, err.Error(), "internal server error", "error should include response body")
}

func TestGetSettings_InvalidJSON(t *testing.T) {
	server := testServerHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	})
	defer server.Close()

	svc := newSettingsService(server.URL, "test-service", "test-host", "test-token")
	settings, err := svc.getSettings(context.Background())

	assert.Nil(t, settings)
	require.Error(t, err)
	assert.NotErrorIs(t, err, config.ErrInvalidServiceKey, "should not be auth error")
	assert.Contains(t, err.Error(), "failed to unmarshal response", "error should indicate JSON parsing failure")
}

func TestGetSettings_NetworkError(t *testing.T) {
	// Using an invalid URL to simulate network error
	svc := newSettingsService("http://invalid-host-that-does-not-exist:9999", "test-service", "test-host", "test-token")
	settings, err := svc.getSettings(context.Background())

	assert.Nil(t, settings)
	require.Error(t, err)
	assert.NotErrorIs(t, err, config.ErrInvalidServiceKey, "should not be auth error")
	assert.Contains(t, err.Error(), "failed to execute request", "error should indicate request execution failure")
}

func TestGetSettings_EmptyResponse(t *testing.T) {
	server := testServerWithResponse(t, http.StatusOK, map[string]interface{}{})
	defer server.Close()

	svc := newSettingsService(server.URL, "test-service", "test-host", "test-token")
	settings, err := svc.getSettings(context.Background())

	require.NoError(t, err)
	require.NotNil(t, settings)
	assert.Equal(t, "", settings.Flags)
	assert.Equal(t, int64(0), settings.Value)
}
