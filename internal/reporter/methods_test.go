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

package reporter

import (
	"context"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/reporter/mocks"
	"testing"

	"github.com/pkg/errors"
	collector "github.com/solarwinds/apm-proto/go/collectorpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostEventsMethod(t *testing.T) {
	host.Start()
	pe := newPostEventsMethod(
		"test-ket",
		[][]byte{
			[]byte("hello"),
			[]byte("world"),
		})
	assert.Equal(t, "PostEvents", pe.String())
	assert.Equal(t, true, pe.RetryOnErr(errConnStale))
	assert.Equal(t, false, pe.RetryOnErr(errRequestTooBig))
	assert.EqualValues(t, 2, pe.MessageLen())

	result := &collector.MessageResult{}
	mockTC := &mocks.TraceCollectorClient{}
	mockTC.On("PostEvents", mock.Anything, mock.Anything).
		Return(result, nil)

	err := pe.Call(context.Background(), mockTC)
	assert.Nil(t, err)
	code, err := pe.ResultCode()
	assert.Equal(t, collector.ResultCode_OK, code)
	assert.Nil(t, err)
	assert.Equal(t, "", pe.Arg())
}

func TestPostMetricsMethod(t *testing.T) {
	pe := newPostMetricsMethod(
		"test-ket",
		[][]byte{
			[]byte("hello"),
			[]byte("world"),
		})
	assert.Equal(t, "PostMetrics", pe.String())
	assert.Equal(t, true, pe.RetryOnErr(errConnStale))
	assert.Equal(t, false, pe.RetryOnErr(errRequestTooBig))
	assert.EqualValues(t, 2, pe.MessageLen())

	result := &collector.MessageResult{}
	mockTC := &mocks.TraceCollectorClient{}
	mockTC.On("PostMetrics", mock.Anything, mock.Anything).
		Return(result, nil)

	err := pe.Call(context.Background(), mockTC)
	assert.Nil(t, err)
	code, err := pe.ResultCode()
	assert.Equal(t, collector.ResultCode_OK, code)
	assert.Nil(t, err)
	assert.Equal(t, "", pe.Arg())
}

func TestPostStatusMethod(t *testing.T) {
	pe := newPostStatusMethod(
		"test-ket",
		[][]byte{
			[]byte("hello"),
			[]byte("world"),
		})
	assert.Equal(t, "PostStatus", pe.String())
	assert.Equal(t, true, pe.RetryOnErr(errConnStale))
	assert.Equal(t, false, pe.RetryOnErr(errRequestTooBig))
	assert.EqualValues(t, 2, pe.MessageLen())

	result := &collector.MessageResult{}
	mockTC := &mocks.TraceCollectorClient{}
	mockTC.On("PostStatus", mock.Anything, mock.Anything).
		Return(result, nil)

	err := pe.Call(context.Background(), mockTC)
	assert.Nil(t, err)
	code, err := pe.ResultCode()
	assert.Nil(t, err)
	assert.Equal(t, collector.ResultCode_OK, code)
	assert.Equal(t, "", pe.Arg())
}

func TestGetSettingsMethod(t *testing.T) {
	pe := newGetSettingsMethod("test-ket")
	assert.Equal(t, "GetSettings", pe.String())
	assert.Equal(t, true, pe.RetryOnErr(errConnStale))
	assert.Equal(t, true, pe.RetryOnErr(errRequestTooBig))
	assert.EqualValues(t, 0, pe.MessageLen())

	result := &collector.SettingsResult{}
	mockTC := &mocks.TraceCollectorClient{}
	mockTC.On("GetSettings", mock.Anything, mock.Anything).
		Return(result, nil)

	err := pe.Call(context.Background(), mockTC)
	assert.Nil(t, err)
	code, err := pe.ResultCode()
	assert.Equal(t, collector.ResultCode_OK, code)
	assert.Nil(t, err)
	assert.Equal(t, "", pe.Arg())
}

func TestPingMethod(t *testing.T) {
	pe := newPingMethod("test-ket", "testConn")
	assert.Equal(t, "Ping testConn", pe.String())
	assert.Equal(t, false, pe.RetryOnErr(errConnStale))
	assert.Equal(t, false, pe.RetryOnErr(errRequestTooBig))
	assert.EqualValues(t, 0, pe.MessageLen())

	result := &collector.MessageResult{}
	mockTC := &mocks.TraceCollectorClient{}
	mockTC.On("Ping", mock.Anything, mock.Anything).
		Return(result, nil)

	err := pe.Call(context.Background(), mockTC)
	assert.Nil(t, err)
	code, err := pe.ResultCode()
	assert.Equal(t, collector.ResultCode_OK, code)
	assert.Nil(t, err)
	assert.Equal(t, "", pe.Arg())
}

func TestGenericMethod(t *testing.T) {
	// test CallSummary before making the RPC call
	pe := newPingMethod("test-ket", "testConn")
	assert.Contains(t, pe.CallSummary(), errRPCNotIssued.Error())

	// test CallSummary when the RPC call fails
	mockTC := &mocks.TraceCollectorClient{}
	err := errors.New("err connection aborted")
	mockTC.On("Ping", mock.Anything, mock.Anything).
		Return(nil, err)
	_ = pe.Call(context.Background(), mockTC)
	assert.Contains(t, pe.CallSummary(), err.Error())

}
