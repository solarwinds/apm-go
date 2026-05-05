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

package swo

import (
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	otelconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestFilterSchemaURLConflict(t *testing.T) {
	t.Run("only conflict returns nil", func(t *testing.T) {
		result := filterSchemaURLConflict(resource.ErrSchemaURLConflict)
		require.Nil(t, result)
	})

	t.Run("conflict joined with other error strips conflict", func(t *testing.T) {
		otherErr := errors.New("some failure")
		combined := errors.Join(resource.ErrSchemaURLConflict, otherErr)
		result := filterSchemaURLConflict(combined)
		require.ErrorContains(t, result, "some failure")
		require.False(t, errors.Is(result, resource.ErrSchemaURLConflict))
	})

	t.Run("other error joined with conflict strips conflict", func(t *testing.T) {
		otherErr := errors.New("some failure")
		combined := errors.Join(otherErr, resource.ErrSchemaURLConflict)
		result := filterSchemaURLConflict(combined)
		require.ErrorContains(t, result, "some failure")
		require.False(t, errors.Is(result, resource.ErrSchemaURLConflict))
	})

	t.Run("multiple non-conflict errors all retained", func(t *testing.T) {
		err1 := errors.New("failure one")
		err2 := errors.New("failure two")
		combined := errors.Join(resource.ErrSchemaURLConflict, err1, err2)
		result := filterSchemaURLConflict(combined)
		require.ErrorContains(t, result, "failure one")
		require.ErrorContains(t, result, "failure two")
		require.False(t, errors.Is(result, resource.ErrSchemaURLConflict))
	})

	t.Run("nil returns nil", func(t *testing.T) {
		result := filterSchemaURLConflict(nil)
		require.Nil(t, result)
	})
}

func TestCreateResourceContainsHostName(t *testing.T) {
	// Disable irrelevant expensive detectors to speed up the test.
	t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", "ec2,azurevm,uams")
	r, err := createResource()
	require.NoError(t, err)
	require.NotNil(t, r)
	val, ok := r.Set().Value(otelconv.HostNameKey)
	require.True(t, ok, "resource must contain host.name attribute")
	hostname, err := os.Hostname()
	require.NoError(t, err)
	require.Equal(t, hostname, val.AsString())
}

// TestCreateResourceServiceName tests the service.name detection logic in createResource.
func TestCreateResourceServiceName(t *testing.T) {
	// Disable irrelevant expensive detectors to speed up the tests.
	const disableDetectors = "ec2,azurevm,uams"
	const validKey = "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:my-service-name"

	t.Run("sets OTEL_SERVICE_NAME from service key when unset", func(t *testing.T) {
		t.Cleanup(func() { config.Load() })
		t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", disableDetectors)
		t.Setenv("SW_APM_SERVICE_KEY", validKey)
		t.Setenv("OTEL_SERVICE_NAME", "")
		config.Load()
		r, err := createResource()
		require.NoError(t, err)
		require.NotNil(t, r)
		require.Equal(t, "my-service-name", os.Getenv(config.EnvOtelServiceNameKey))
		val, ok := r.Set().Value(otelconv.ServiceNameKey)
		require.True(t, ok, "resource must contain service.name attribute")
		require.Equal(t, "my-service-name", val.AsString())
	})

	t.Run("does not override OTEL_SERVICE_NAME when already set", func(t *testing.T) {
		t.Cleanup(func() { config.Load() })
		t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", disableDetectors)
		t.Setenv("SW_APM_SERVICE_KEY", validKey)
		t.Setenv("OTEL_SERVICE_NAME", "envvar-service-name")
		config.Load()
		r, err := createResource()
		require.NoError(t, err)
		require.NotNil(t, r)
		require.Equal(t, "envvar-service-name", os.Getenv(config.EnvOtelServiceNameKey))
		val, ok := r.Set().Value(otelconv.ServiceNameKey)
		require.True(t, ok, "resource must contain service.name attribute")
		require.Equal(t, "envvar-service-name", val.AsString())
	})

	t.Run("leaves OTEL_SERVICE_NAME unset when no valid service key", func(t *testing.T) {
		t.Cleanup(func() { config.Load() })
		t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", disableDetectors)
		t.Setenv("SW_APM_SERVICE_KEY", "")
		t.Setenv("OTEL_SERVICE_NAME", "")
		config.Load()
		r, err := createResource()
		require.NoError(t, err)
		require.NotNil(t, r)
		require.Equal(t, "", os.Getenv(config.EnvOtelServiceNameKey))
		val, ok := r.Set().Value(otelconv.ServiceNameKey)
		require.True(t, ok, "resource must contain service.name attribute")
		require.Contains(t, val.AsString(), "unknown_service")
	})
}

func TestCreateResourceContainsServiceInstanceID(t *testing.T) {
	// Disable irrelevant expensive detectors to speed up the test.
	t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", "ec2,azurevm,uams")
	r, err := createResource()
	require.NoError(t, err)
	require.NotNil(t, r)
	val, ok := r.Set().Value(otelconv.ServiceInstanceIDKey)
	require.True(t, ok, "resource must contain service.instance.id attribute")
	_, err = uuid.Parse(val.AsString())
	require.NoError(t, err, "service.instance.id must be a valid UUID")
}

func TestCreateResourceErrSchemaURLConflictNotReturned(t *testing.T) {
	// Disable irrelevant expensive detectors to speed up the test.
	t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", "ec2,azurevm,uams")
	// ErrSchemaURLConflict is non-fatal and must never be propagated to the caller.
	r, err := createResource()
	require.NotNil(t, r)
	require.False(t, errors.Is(err, resource.ErrSchemaURLConflict))
}
