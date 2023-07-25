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

package swotel

import (
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestGetSetSw(t *testing.T) {
	var err error
	ts := trace.TraceState{}
	ts, err = ts.Insert("rojo", "00f067aa0ba902b7")
	require.NoError(t, err)
	require.Equal(t, "", GetSw(ts))

	ts, err = SetSw(ts, "foo bar")
	require.NoError(t, err)
	require.Equal(t, "foo bar", GetSw(ts))
	require.Equal(t, "00f067aa0ba902b7", ts.Get("rojo"))
	require.Equal(t, "foo bar", ts.Get("sw"))
}

func TestInternalState(t *testing.T) {
	var err error
	var st string
	var ts trace.TraceState
	ts = trace.TraceState{}
	ts, err = ts.Insert("rojo", "00f067aa0ba902b7")
	require.NoError(t, err)
	st, err = GetInternalState(ts, XTraceOptResp)
	require.Equal(t, "", st)
	require.NoError(t, err)
	ts, err = SetInternalState(ts, XTraceOptResp, "foo=bar")
	require.NoError(t, err)
	require.Equal(t, "foo####bar", ts.Get("xtrace_options_response"))
	st, err = GetInternalState(ts, XTraceOptResp)
	require.NoError(t, err)
	require.Equal(t, "foo=bar", st)

	// Delete
	ts, err = RemoveInternalState(ts, XTraceOptResp)
	require.NoError(t, err)
	require.Equal(t, "", ts.Get("xtrace_options_response"))

	// Test invalid internal keys
	ts = trace.TraceState{}
	ts, err = ts.Insert("rojo", "00f067aa0ba902b7")
	require.NoError(t, err)
	require.Equal(t, "rojo=00f067aa0ba902b7", ts.String())

	// This is why iotas are inferior to enums
	st, err = GetInternalState(ts, 123)
	require.Error(t, err)
	require.Equal(t, "", st)
	require.Equal(t, "invalid key: 123", err.Error())

	ts, err = SetInternalState(ts, 123, "must not be set")
	require.Error(t, err)
	require.Equal(t, "invalid key: 123", err.Error())
	require.Equal(t, "rojo=00f067aa0ba902b7", ts.String())

	ts, err = RemoveInternalState(ts, 123)
	require.Error(t, err)
	require.Equal(t, "invalid key: 123", err.Error())
	require.Equal(t, "rojo=00f067aa0ba902b7", ts.String())
}
