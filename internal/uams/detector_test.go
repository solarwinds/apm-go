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
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/solarwinds/apm-go/internal/testutils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
)

func resourceAttrs(r *resource.Resource) map[string]string {
	attrs := make(map[string]string)
	for _, kv := range r.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}
	return attrs
}

func TestDetectFileExistsHttpOK(t *testing.T) {
	fileUUID, err := uuid.NewRandom()
	require.NoError(t, err)
	httpUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	filePath := testutils.WriteUUIDFile(t, fileUUID)
	svr := testutils.Srv(t, testutils.UamsClientResponse(httpUUID), http.StatusOK)
	defer svr.Close()
	overrideUamsFile(t, filePath)
	overrideUamsURL(t, svr.URL)

	res, err := New().Detect(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
	attrs := resourceAttrs(res)
	require.Equal(t, fileUUID.String(), attrs[constants.UamsClientIdAttribute])
	require.Equal(t, fileUUID.String(), attrs["host.id"])
}

func TestDetectFileMissingHttpOK(t *testing.T) {
	httpUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	svr := testutils.Srv(t, testutils.UamsClientResponse(httpUUID), http.StatusOK)
	defer svr.Close()
	overrideUamsFile(t, nonexistentFile)
	overrideUamsURL(t, svr.URL)

	res, err := New().Detect(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
	attrs := resourceAttrs(res)
	require.Equal(t, httpUUID.String(), attrs[constants.UamsClientIdAttribute])
	require.Equal(t, httpUUID.String(), attrs["host.id"])
}

func TestDetectFileExistsHttpError(t *testing.T) {
	fileUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	filePath := testutils.WriteUUIDFile(t, fileUUID)
	svr := testutils.Srv(t, "", http.StatusInternalServerError)
	defer svr.Close()
	overrideUamsFile(t, filePath)
	overrideUamsURL(t, svr.URL)

	res, err := New().Detect(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
	attrs := resourceAttrs(res)
	require.Equal(t, fileUUID.String(), attrs[constants.UamsClientIdAttribute])
	require.Equal(t, fileUUID.String(), attrs["host.id"])
}

func TestDetectFileMissingHttpError(t *testing.T) {
	svr := testutils.Srv(t, "", http.StatusInternalServerError)
	defer svr.Close()
	overrideUamsFile(t, nonexistentFile)
	overrideUamsURL(t, svr.URL)

	res, err := New().Detect(context.Background())
	require.NoError(t, err)
	require.Nil(t, res)
}
