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
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestUpdateClientId(t *testing.T) {
	defer resetState()
	var defaultTime time.Time // default (1970-01-01 00:00:00)
	require.Equal(t, uuid.Nil, currState.clientId)
	require.Equal(t, defaultTime, currState.updated)
	require.Equal(t, "", currState.via)

	require.Equal(t, uuid.Nil, GetCurrentClientId())

	uid, err := uuid.NewRandom()
	require.NoError(t, err)
	a := time.Now()
	updateClientId(uid, "file")
	b := time.Now()
	require.Equal(t, uid, currState.clientId)
	require.Equal(t, "file", currState.via)
	require.True(t, currState.updated.After(a))
	require.True(t, currState.updated.Before(b))

	require.Equal(t, uid, GetCurrentClientId())
}

func determineFileForOS() string {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		return windowsFilePath
	}
	return linuxFilePath
}

func resetState() {
	currState = &state{}
}

func TestClientIdCheckFromFile(t *testing.T) {
	defer resetState()
	f := determineFileForOS()
	require.NoFileExists(t, f, "Test needs to write to file, but it may exist for another purpose", f)
	clientIdCheck()
	require.Equal(t, uuid.Nil, currState.clientId)

	dir := filepath.Dir(f)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// This test will fail if you don't have the path required. See `determineFileForOS` above.
		// For macOS, we use the linuxFilePath, so you'll want to do something like:
		//   sudo mkdir /opt/solarwinds
		//   sudo chown ${USER}:admin /opt/solarwinds
		require.NoError(t, os.MkdirAll(dir, 0755))
	}
	require.DirExists(t, dir)

	uid, err := uuid.NewRandom()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(f, []byte(uid.String()), 0644))

	defer func() {
		require.NoError(t, os.Remove(f))
	}()

	clientIdCheck()

	require.Equal(t, uid, currState.clientId)
	require.Equal(t, "file", currState.via)
}

func TestClientIdCheckFromHttp(t *testing.T) {
	defer resetState()
	f := determineFileForOS()
	require.NoFileExists(t, f, "Test needs to write to file, but it may exist for another purpose", f)

	clientIdCheck()
	require.Equal(t, uuid.Nil, currState.clientId)

	uid, err := uuid.NewRandom()
	require.NoError(t, err)
	server := &http.Server{Addr: ":2113"}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprintf(w, `{"uamsclient_id": "%s"}`, uid.String())
		require.NoError(t, err)
	})
	http.Handle("/info/uamsclient", handler)
	go func() {
		_ = server.ListenAndServe()
	}()

	defer func() {
		require.NoError(t, server.Shutdown(context.Background()))
	}()

	time.Sleep(10 * time.Millisecond)
	clientIdCheck()

	require.Equal(t, uid, currState.clientId)
	require.Equal(t, "http", currState.via)
}
