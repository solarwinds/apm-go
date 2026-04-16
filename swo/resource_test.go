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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
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

func TestCreateResourceErrSchemaURLConflictNotReturned(t *testing.T) {
	// Disable network-dependent detectors to keep the test fast and hermetic.
	t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", "ec2,azurevm,uams")
	// ErrSchemaURLConflict is non-fatal and must never be propagated to the caller.
	r, err := createResource()
	require.NotNil(t, r)
	require.False(t, errors.Is(err, resource.ErrSchemaURLConflict))
}
