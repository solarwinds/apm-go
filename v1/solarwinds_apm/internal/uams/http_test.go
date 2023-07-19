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
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadFromHttpConnRefused(t *testing.T) {
	uid, err := ReadFromHttp("http://localhost:12345")
	require.Error(t, err)
	require.Equal(t, uuid.Nil, uid)
	require.Equal(t, `Get "http://localhost:12345": dial tcp 127.0.0.1:12345: connect: connection refused`, err.Error())
}

func srv(t *testing.T, response string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, err := fmt.Fprint(w, response)
		require.NoError(t, err)
	}))
}

func TestReadFromHttp(t *testing.T) {
	expectedUid, err := uuid.NewRandom()
	require.NoError(t, err)
	response := fmt.Sprintf(
		`{
    "is_registered": true,
    "otel_endpoint_access": false,
    "usc_connectivity": true,
    "uamsclient_id": "%s"
}`,
		expectedUid.String(),
	)
	svr := srv(t, response, http.StatusOK)
	defer svr.Close()

	uid, err := ReadFromHttp(svr.URL)
	require.NoError(t, err)
	require.Equal(t, expectedUid, uid)
}

func TestReadFromHttpInvalidIDType(t *testing.T) {
	response := `{
    "is_registered": true,
    "otel_endpoint_access": false,
    "usc_connectivity": true,
    "uamsclient_id": 123456 
}`
	svr := srv(t, response, http.StatusOK)
	defer svr.Close()

	uid, err := ReadFromHttp(svr.URL)
	require.Error(t, err)
	require.Equal(t, "expected string, got float64 instead", err.Error())
	require.Equal(t, uuid.Nil, uid)
}

func TestReadFromHttpInvalidFormat(t *testing.T) {
	response := `{
    "is_registered": true,
    "otel_endpoint_access": false,
    "usc_connectivity": true,
    "uamsclient_id": "Now is the winter of our discontent!" 
}`
	svr := srv(t, response, http.StatusOK)
	defer svr.Close()

	uid, err := ReadFromHttp(svr.URL)
	require.Error(t, err)
	require.Equal(t, "invalid UUID format", err.Error())
	require.Equal(t, uuid.Nil, uid)
}

func TestReadFromHttpMissingKey(t *testing.T) {
	response :=
		`{
    "is_registered": true,
    "otel_endpoint_access": false,
    "usc_connectivity": true
}`
	svr := srv(t, response, http.StatusOK)
	defer svr.Close()

	uid, err := ReadFromHttp(svr.URL)
	require.Error(t, err)
	require.Equal(t, "uamsclient_id not found", err.Error())
	require.Equal(t, uuid.Nil, uid)
}

func TestReadFromHttpInvalidJSON(t *testing.T) {
	response := "this is not json"
	svr := srv(t, response, http.StatusOK)
	defer svr.Close()

	uid, err := ReadFromHttp(svr.URL)
	require.Error(t, err)
	require.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	require.Equal(t, uuid.Nil, uid)
}

func TestReadFromHttpInvalidStatus(t *testing.T) {
	response := "foo bar baz"
	svr := srv(t, response, http.StatusInternalServerError)
	defer svr.Close()

	uid, err := ReadFromHttp(svr.URL)
	require.Error(t, err)
	require.Equal(t, "uamsclient: expected 200 status code, got 500 Internal Server Error", err.Error())
	require.Equal(t, uuid.Nil, uid)
}
