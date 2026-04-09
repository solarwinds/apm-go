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

package uams

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestReadUamsClientId_WhenUnavailable verifies that readUamsClientId returns
// uuid.Nil and a non-nil error when neither the UAMS client file nor the HTTP
// endpoint is available, which is the expected state in most test environments.
func TestReadUamsClientIdWhenUnavailable(t *testing.T) {
	id, err := readUamsClientId()

	require.Equal(t, uuid.Nil, id)
	require.Error(t, err)
}
