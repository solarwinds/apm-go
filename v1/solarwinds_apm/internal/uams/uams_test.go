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

package uams

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestUpdateClientId(t *testing.T) {
	var defaultTime time.Time // default (1970-01-01 00:00:00)
	require.Equal(t, uuid.Nil, uamsState.clientId)
	require.Equal(t, defaultTime, uamsState.updated)
	require.Equal(t, "", uamsState.via)

	require.Equal(t, uuid.Nil, GetCurrentClientId())

	uid, err := uuid.NewRandom()
	require.NoError(t, err)
	a := time.Now()
	updateClientId(uid, "file")
	b := time.Now()
	require.Equal(t, uid, uamsState.clientId)
	require.Equal(t, "file", uamsState.via)
	require.True(t, uamsState.updated.After(a))
	require.True(t, uamsState.updated.Before(b))

	require.Equal(t, uid, GetCurrentClientId())
}
