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
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/solarwinds/apm-go/internal/testutils"
	"github.com/stretchr/testify/require"
)

const nonexistentFile = "/tmp/nonexistent-uams-test-file"

// overrideUamsFile temporarily replaces uamsFilePath for the duration of the test.
func overrideUamsFile(t *testing.T, path string) {
	t.Helper()
	orig := uamsFilePath
	uamsFilePath = path
	t.Cleanup(func() { uamsFilePath = orig })
}

// overrideUamsURL temporarily replaces uamsClientURL for the duration of the test.
func overrideUamsURL(t *testing.T, url string) {
	t.Helper()
	orig := uamsClientURL
	uamsClientURL = url
	t.Cleanup(func() { uamsClientURL = orig })
}

func TestReadUamsClientIdFileExistsHttpOK(t *testing.T) {
	fileUUID, err := uuid.NewRandom()
	require.NoError(t, err)
	httpUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	filePath := testutils.WriteUUIDFile(t, fileUUID)
	svr := testutils.Srv(t, testutils.UamsClientResponse(httpUUID), http.StatusOK)
	defer svr.Close()
	overrideUamsFile(t, filePath)
	overrideUamsURL(t, svr.URL)

	id, err := readUamsClientId()
	require.NoError(t, err)
	require.Equal(t, fileUUID, id)
}

func TestReadUamsClientIdFileMissingHttpOK(t *testing.T) {
	httpUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	svr := testutils.Srv(t, testutils.UamsClientResponse(httpUUID), http.StatusOK)
	defer svr.Close()
	overrideUamsFile(t, nonexistentFile)
	overrideUamsURL(t, svr.URL)

	id, err := readUamsClientId()
	require.NoError(t, err)
	require.Equal(t, httpUUID, id)
}

func TestReadUamsClientIdFileExistsHttpError(t *testing.T) {
	fileUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	filePath := testutils.WriteUUIDFile(t, fileUUID)
	svr := testutils.Srv(t, "", http.StatusInternalServerError)
	defer svr.Close()
	overrideUamsFile(t, filePath)
	overrideUamsURL(t, svr.URL)

	id, err := readUamsClientId()
	require.NoError(t, err)
	require.Equal(t, fileUUID, id)
}

func TestReadUamsClientIdFileMissingHttpError(t *testing.T) {
	svr := testutils.Srv(t, "", http.StatusInternalServerError)
	defer svr.Close()
	overrideUamsFile(t, nonexistentFile)
	overrideUamsURL(t, svr.URL)

	id, err := readUamsClientId()
	require.Equal(t, uuid.Nil, id)
	require.Error(t, err)
}
