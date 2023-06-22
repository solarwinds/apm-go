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
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"strings"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/bson"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config"
	g "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/graphtest"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/host"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics"
	pb "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter/collector"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter/mocks"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mbson "gopkg.in/mgo.v2/bson"
)

const TestServiceKey = "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go"

// this runs before init()
var _ = func() (_ struct{}) {
	periodicTasksDisabled = true

	os.Clearenv()
	os.Setenv("SW_APM_SERVICE_KEY", TestServiceKey)
	os.Setenv("SW_APM_DEBUG_LEVEL", "debug")

	config.Load()
	return
}()

// ========================= Test Reporter =============================

func TestReportEvent(t *testing.T) {
	r := SetTestReporter()
	ctx := newTestContext(t)
	assert.Error(t, r.reportEvent(ctx, nil))
	assert.Len(t, r.EventBufs, 0) // no reporting

	// mismatched task IDs
	ev, err := ctx.newEvent(LabelExit, testLayer)
	assert.NoError(t, err)
	assert.Error(t, r.reportEvent(nil, ev))
	assert.Len(t, r.EventBufs, 0) // no reporting

	ctx2 := newTestContext(t)
	e2, err := ctx2.newEvent(LabelEntry, "layer2")
	assert.NoError(t, err)
	assert.Error(t, r.reportEvent(ctx2, ev))
	assert.Error(t, r.reportEvent(ctx, e2))

	// successful event
	assert.NoError(t, r.reportEvent(ctx, ev))
	r.Close(1)
	assert.Len(t, r.EventBufs, 1)

	// re-report: shouldn't work (op IDs the same, reporter closed)
	assert.Error(t, r.reportEvent(ctx, ev))

	g.AssertGraph(t, r.EventBufs, 1, g.AssertNodeMap{
		{"go_test", "exit"}: {},
	})
}

func TestReportMetric(t *testing.T) {
	r := SetTestReporter()
	spanMsg := &metrics.HTTPSpanMessage{
		BaseSpanMessage: metrics.BaseSpanMessage{
			Duration: time.Second,
			HasError: false,
		},
		Transaction: "tname",
		Path:        "/path/to/url",
		Status:      203,
		Method:      "HEAD",
	}
	err := ReportSpan(spanMsg)
	assert.NoError(t, err)
	r.Close(1)
	assert.Len(t, r.SpanMessages, 1)
	sp, ok := r.SpanMessages[0].(*metrics.HTTPSpanMessage)
	require.True(t, ok)
	require.NotNil(t, sp)
	assert.True(t, reflect.DeepEqual(spanMsg, sp))
}

// test behavior of the TestReporter
func TestTestReporter(t *testing.T) {
	r := SetTestReporter()
	r.Close(0) // wait on event that will never be reported: causes timeout
	assert.Len(t, r.EventBufs, 0)

	r = SetTestReporter()
	go func() { // simulate late event
		time.Sleep(100 * time.Millisecond)
		ctx := newTestContext(t)
		ev, err := ctx.newEvent(LabelExit, testLayer)
		assert.NoError(t, err)
		assert.NoError(t, r.reportEvent(ctx, ev))
	}()
	r.Close(1) // wait on late event -- blocks until timeout or event received
	assert.Len(t, r.EventBufs, 1)

	// send an event after calling Close -- should panic
	assert.Panics(t, func() {
		ctx := newTestContext(t)
		ev, err := ctx.newEvent(LabelExit, testLayer)
		assert.NoError(t, err)
		assert.NoError(t, r.reportEvent(ctx, ev))
	})
}

// ========================= NULL Reporter =============================

func TestNullReporter(t *testing.T) {
	nullR := &nullReporter{}
	assert.NoError(t, nullR.reportEvent(nil, nil))
	assert.NoError(t, nullR.reportStatus(nil, nil))
	assert.NoError(t, nullR.reportSpan(nil))

}

// ========================= GRPC Reporter =============================

func TestGRPCReporter(t *testing.T) {
	// start test gRPC server
	os.Setenv("SW_APM_DEBUG_LEVEL", "debug")
	config.Load()
	addr := "localhost:4567"
	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	// set gRPC reporter
	os.Setenv("SW_APM_COLLECTOR", addr)
	os.Setenv("SW_APM_TRUSTEDPATH", testCertFile)
	config.Load()
	oldReporter := globalReporter
	setGlobalReporter("ssl")

	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)

	// Test WaitForReady
	// The reporter is not ready when there is no default setting.
	ctxTm1, cancel1 := context.WithTimeout(context.Background(), 0)
	defer cancel1()
	assert.False(t, r.WaitForReady(ctxTm1))

	// The reporter becomes ready after it has got the default setting.
	ready := make(chan bool, 1)
	r.getSettings(ready)
	ctxTm2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel2()
	assert.True(t, r.WaitForReady(ctxTm2))
	assert.True(t, r.isReady())

	ctx := newTestContext(t)
	ev1, err := ctx.newEvent(LabelInfo, "layer1")
	assert.NoError(t, err)
	ev2, err := ctx.newEvent(LabelInfo, "layer2")
	assert.NoError(t, err)

	assert.Error(t, r.reportEvent(nil, nil))
	assert.Error(t, r.reportEvent(ctx, nil))
	assert.NoError(t, r.reportEvent(ctx, ev1))

	assert.Error(t, r.reportStatus(nil, nil))
	assert.Error(t, r.reportStatus(ctx, nil))
	// time.Sleep(time.Second)
	assert.NoError(t, r.reportStatus(ctx, ev2))

	assert.Equal(t, addr, r.conn.address)

	assert.Equal(t, TestServiceKey, r.serviceKey.Load())

	assert.Equal(t, int32(grpcMetricIntervalDefault), r.collectMetricInterval)
	assert.Equal(t, grpcGetSettingsIntervalDefault, r.getSettingsInterval)
	assert.Equal(t, grpcSettingsTimeoutCheckIntervalDefault, r.settingsTimeoutCheckInterval)

	time.Sleep(time.Second)

	// The reporter becomes not ready after the default setting has been deleted
	removeSetting("")
	r.checkSettingsTimeout(make(chan bool, 1))

	assert.False(t, r.isReady())
	ctxTm3, cancel3 := context.WithTimeout(context.Background(), 0)
	assert.False(t, r.WaitForReady(ctxTm3))
	defer cancel3()

	// stop test reporter
	server.Stop()
	globalReporter = oldReporter

	// assert data received
	require.Len(t, server.events, 1)
	assert.Equal(t, server.events[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.events[0].Messages, 1)

	require.Len(t, server.status, 1)
	assert.Equal(t, server.status[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.status[0].Messages, 1)

	dec1, dec2 := mbson.M{}, mbson.M{}
	err = mbson.Unmarshal(server.events[0].Messages[0], &dec1)
	require.NoError(t, err)
	err = mbson.Unmarshal(server.status[0].Messages[0], &dec2)
	require.NoError(t, err)

	assert.Equal(t, dec1["Layer"], "layer1")
	assert.Equal(t, dec1["Hostname"], host.Hostname())
	assert.Equal(t, dec1["Label"], LabelInfo)
	assert.Equal(t, dec1["PID"], host.PID())

	assert.Equal(t, dec2["Layer"], "layer2")
}

func TestShutdownGRPCReporter(t *testing.T) {
	// start test gRPC server
	os.Setenv("SW_APM_DEBUG_LEVEL", "debug")
	addr := "localhost:4567"
	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	// set gRPC reporter
	os.Setenv("SW_APM_COLLECTOR", addr)
	os.Setenv("SW_APM_TRUSTEDPATH", testCertFile)
	config.Load()
	oldReporter := globalReporter
	// numGo := runtime.NumGoroutine()
	setGlobalReporter("ssl")

	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)
	r.ShutdownNow()

	assert.Equal(t, true, r.Closed())

	// // Print current goroutines stack
	// buf := make([]byte, 1<<16)
	// runtime.Stack(buf, true)
	// fmt.Printf("%s", buf)

	e := r.ShutdownNow()
	assert.NotEqual(t, nil, e)

	// stop test reporter
	server.Stop()
	globalReporter = oldReporter
	// fmt.Println(buf)
}

func TestInvalidKey(t *testing.T) {
	var buf utils.SafeBuffer
	var writers []io.Writer

	writers = append(writers, &buf)
	writers = append(writers, os.Stderr)

	log.SetOutput(io.MultiWriter(writers...))

	defer func() {
		log.SetOutput(os.Stderr)
	}()

	invalidKey := "invalidf6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:Go"
	os.Setenv("SW_APM_DEBUG_LEVEL", "debug")
	oldKey := os.Getenv("SW_APM_SERVICE_KEY")
	os.Setenv("SW_APM_SERVICE_KEY", invalidKey)
	addr := "localhost:4567"
	os.Setenv("SW_APM_COLLECTOR", addr)
	os.Setenv("SW_APM_TRUSTEDPATH", testCertFile)

	// start test gRPC server
	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	// set gRPC reporter
	config.Load()
	oldReporter := globalReporter

	log.SetLevel(log.INFO)
	setGlobalReporter("ssl")
	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)
	ctx := newTestContext(t)
	ev1, _ := ctx.newEvent(LabelInfo, "hello-from-invalid-key")
	assert.NoError(t, r.reportEvent(ctx, ev1))

	time.Sleep(time.Second)

	// The agent reporter should be closed due to received INVALID_API_KEY from the collector
	assert.Equal(t, true, r.Closed())

	e := r.ShutdownNow()
	assert.NotEqual(t, nil, e)

	// Tear down everything.
	server.Stop()
	globalReporter = oldReporter
	os.Setenv("SW_APM_SERVICE_KEY", oldKey)

	patterns := []string{
		"rsp=INVALID_API_KEY",
		"Shutting down the reporter",
		// "periodicTasks goroutine exiting",
		"eventSender goroutine exiting",
		"spanMessageAggregator goroutine exiting",
		"statusSender goroutine exiting",
		"eventBatchSender goroutine exiting",
	}
	for _, ptn := range patterns {
		assert.True(t, strings.Contains(buf.String(), ptn), buf.String()+"^^^^^^"+ptn)
	}
	log.SetLevel(log.WARNING)
}

func TestDefaultBackoff(t *testing.T) {
	var backoff []int64
	expected := []int64{
		500, 750, 1125, 1687, 2531, 3796, 5695, 8542, 12814, 19221, 28832,
		43248, 60000, 60000, 60000, 60000, 60000, 60000, 60000, 60000}
	bf := func(d time.Duration) { backoff = append(backoff, d.Nanoseconds()/1e6) }
	for i := 1; i <= grpcMaxRetries+1; i++ {
		DefaultBackoff(i, bf)
	}
	assert.Equal(t, expected, backoff)
	assert.NotNil(t, DefaultBackoff(grpcMaxRetries+1, func(d time.Duration) {}))
}

type NoopDialer struct{}

func (d *NoopDialer) Dial(p DialParams) (*grpc.ClientConn, error) {
	return nil, nil
}

func TestInvokeRPC(t *testing.T) {
	var buf utils.SafeBuffer
	var writers []io.Writer

	writers = append(writers, &buf)
	writers = append(writers, os.Stderr)

	log.SetOutput(io.MultiWriter(writers...))

	defer func() {
		log.SetOutput(os.Stderr)
	}()

	c := &grpcConnection{
		name:        "events channel",
		client:      nil,
		connection:  nil,
		address:     "test-addr",
		certificate: "",
		queueStats:  &metrics.EventQueueStats{},
		backoff: func(retries int, wait func(d time.Duration)) error {
			if retries > grpcMaxRetries {
				return errGiveUpAfterRetries
			}
			return nil
		},
		Dialer:      &NoopDialer{},
		flushed:     make(chan struct{}),
		maxReqBytes: 6 * 1024 * 1024,
	}
	_ = c.connect()

	// Test reporter exiting
	mockMethod := &mocks.Method{}
	mockMethod.On("String").Return("mock")
	mockMethod.On("ServiceKey").Return("")
	mockMethod.On("Call", mock.Anything, mock.Anything).
		Return(nil)
	mockMethod.On("Message").Return(nil)
	mockMethod.On("MessageLen").Return(int64(0))
	mockMethod.On("RequestSize").Return(int64(1))
	mockMethod.On("CallSummary").Return("summary")
	mockMethod.On("Arg").Return("testArg")

	exit := make(chan struct{})
	close(exit)
	c.setFlushed()
	assert.Equal(t, errReporterExiting, c.InvokeRPC(exit, mockMethod))

	// Test invalid service key
	exit = make(chan struct{})
	mockMethod = &mocks.Method{}
	mockMethod.On("Call", mock.Anything, mock.Anything).
		Return(nil)
	mockMethod.On("String").Return("mock")
	mockMethod.On("ServiceKey").Return("serviceKey")
	mockMethod.On("Message").Return(nil)
	mockMethod.On("MessageLen").Return(int64(0))
	mockMethod.On("RequestSize").Return(int64(1))
	mockMethod.On("CallSummary").Return("summary")
	mockMethod.On("Arg").Return("testArg")
	mockMethod.On("ResultCode", mock.Anything, mock.Anything).
		Return(pb.ResultCode_INVALID_API_KEY, nil)
	mockMethod.On("RetryOnErr", mock.Anything).Return(false)

	assert.Equal(t, errInvalidServiceKey, c.InvokeRPC(exit, mockMethod))

	// Test no retry
	mockMethod = &mocks.Method{}
	mockMethod.On("Call", mock.Anything, mock.Anything).
		Return(nil)
	mockMethod.On("String").Return("mock")
	mockMethod.On("ServiceKey").Return("serviceKey")
	mockMethod.On("Message").Return(nil)
	mockMethod.On("MessageLen").Return(int64(0))
	mockMethod.On("RequestSize").Return(int64(1))
	mockMethod.On("CallSummary").Return("summary")
	mockMethod.On("Arg").Return("testArg")
	mockMethod.On("ResultCode", mock.Anything, mock.Anything).
		Return(pb.ResultCode_LIMIT_EXCEEDED, nil)

	mockMethod.On("RetryOnErr", mock.Anything).Return(false)
	assert.Equal(t, errNoRetryOnErr, c.InvokeRPC(exit, mockMethod))

	// Test invocation error / recovery logs
	failsNum := grpcRetryLogThreshold + (grpcMaxRetries-grpcRetryLogThreshold)/2

	mockMethod = &mocks.Method{}
	mockMethod.On("String").Return("mock")
	mockMethod.On("ServiceKey").Return("serviceKey")
	mockMethod.On("Message").Return(nil)
	mockMethod.On("MessageLen").Return(int64(0))
	mockMethod.On("RequestSize").Return(int64(1))
	mockMethod.On("CallSummary").Return("summary")
	mockMethod.On("Arg").Return("testArg")
	mockMethod.On("RetryOnErr", mock.Anything).Return(true)
	mockMethod.On("ResultCode", mock.Anything, mock.Anything).
		Return(pb.ResultCode_OK, nil)

	mockMethod.On("Call", mock.Anything, mock.Anything).
		Return(func(ctx context.Context, c pb.TraceCollectorClient) error {
			failsNum--
			if failsNum <= 0 {
				return nil
			} else {
				return status.Error(codes.Canceled, "Canceled")
			}
		})
	assert.Equal(t, nil, c.InvokeRPC(exit, mockMethod))
	assert.True(t, strings.Contains(buf.String(), "invocation error"))
	assert.True(t, strings.Contains(buf.String(), "error recovered"))

	// Test redirect
	redirectNum := 1
	mockMethod = &mocks.Method{}
	mockMethod.On("String").Return("mock")
	mockMethod.On("ServiceKey").Return("serviceKey")
	mockMethod.On("Message").Return(nil)
	mockMethod.On("MessageLen").Return(int64(0))
	mockMethod.On("RequestSize").Return(int64(1))
	mockMethod.On("CallSummary").Return("summary")
	mockMethod.On("RetryOnErr", mock.Anything).Return(true)
	mockMethod.On("Arg", mock.Anything, mock.Anything).
		Return("new-addr:9999")
	mockMethod.On("ResultCode", mock.Anything, mock.Anything).
		Return(func() pb.ResultCode {
			redirectNum--
			if redirectNum < 0 {
				return pb.ResultCode_OK
			} else {
				return pb.ResultCode_REDIRECT
			}
		}, nil)

	mockMethod.On("Call", mock.Anything, mock.Anything).
		Return(nil)
	assert.Equal(t, nil, c.InvokeRPC(exit, mockMethod))
	assert.True(t, c.isActive())
	assert.Equal(t, "new-addr:9999", c.address)

	// Test request too big
	mockMethod = &mocks.Method{}
	mockMethod.On("Call", mock.Anything, mock.Anything).
		Return(nil)
	mockMethod.On("String").Return("mock")
	mockMethod.On("ServiceKey").Return("serviceKey")
	mockMethod.On("Message").Return(nil)
	mockMethod.On("MessageLen").Return(int64(0))
	mockMethod.On("RequestSize").Return(int64(6*1024*1024 + 1))
	mockMethod.On("CallSummary").Return("summary")
	mockMethod.On("Arg").Return("testArg")
	mockMethod.On("RetryOnErr", mock.Anything).Return(false)

	assert.Contains(t, c.InvokeRPC(exit, mockMethod).Error(), errNoRetryOnErr.Error())
}

func TestInitReporter(t *testing.T) {
	// Test disable agent
	os.Setenv("SW_APM_DISABLED", "true")
	config.Load()
	initReporter()
	require.IsType(t, &nullReporter{}, globalReporter)

	// Test enable agent
	os.Unsetenv("SW_APM_DISABLED")
	os.Setenv("SW_APM_REPORTER", "ssl")
	config.Load()
	assert.False(t, config.GetDisabled())

	initReporter()
	require.IsType(t, &grpcReporter{}, globalReporter)
}

func TestCollectMetricsNextInterval(t *testing.T) {
	r := &grpcReporter{collectMetricInterval: 10}
	next := r.collectMetricsNextInterval()
	// very weak check
	assert.True(t, next <= time.Second*10, next)
}

func TestCustomMetrics(t *testing.T) {
	r := &grpcReporter{
		// Other fields are not needed.
		customMetrics: metrics.NewMeasurements(true, grpcMetricIntervalDefault, 500),
	}

	// Test non-positive count
	assert.NotNil(t, r.CustomSummaryMetric("Summary", 1.1, metrics.MetricOptions{
		Count:   0,
		HostTag: true,
		Tags:    map[string]string{"hello": "world"},
	}))
	assert.NotNil(t, r.CustomIncrementMetric("Incremental", metrics.MetricOptions{
		Count:   -1,
		HostTag: true,
		Tags:    map[string]string{"hi": "globe"},
	}))

	r.CustomSummaryMetric("Summary", 1.1, metrics.MetricOptions{
		Count:   1,
		HostTag: true,
		Tags:    map[string]string{"hello": "world"},
	})
	r.CustomIncrementMetric("Incremental", metrics.MetricOptions{
		Count:   1,
		HostTag: true,
		Tags:    map[string]string{"hi": "globe"},
	})
	custom := metrics.BuildMessage(r.customMetrics.CopyAndReset(grpcMetricIntervalDefault), false)

	bbuf := bson.WithBuf(custom)
	mMap := mbson.M{}
	mbson.Unmarshal(bbuf.GetBuf(), mMap)

	assert.Equal(t, mMap["IsCustom"], true)
	assert.NotEqual(t, mMap["Timestamp_u"], 0)
	assert.Equal(t, mMap["MetricsFlushInterval"], grpcMetricIntervalDefault)
	assert.NotEqual(t, mMap["IPAddresses"], nil)
	assert.NotEqual(t, mMap["Distro"], "")

	mts := mMap["measurements"].([]interface{})
	require.Equal(t, len(mts), 2)

	mSummary := mts[0].(mbson.M)
	mIncremental := mts[1].(mbson.M)

	if mSummary["name"] == "Incremental" {
		mSummary, mIncremental = mIncremental, mSummary
	}

	assert.Equal(t, mSummary["name"], "Summary")
	assert.Equal(t, mSummary["count"], 1)
	assert.Equal(t, mSummary["sum"], 1.1)
	assert.EqualValues(t, mbson.M{"hello": "world"}, mSummary["tags"])

	assert.Equal(t, mIncremental["name"], "Incremental")
	assert.Equal(t, mIncremental["count"], 1)
	assert.EqualValues(t, mbson.M{"hi": "globe"}, mIncremental["tags"])
}

// testProxy performs tests of http/https proxy.
func testProxy(t *testing.T, proxyUrl string) {
	addr := "localhost:4567"

	os.Setenv("SW_APM_DEBUG_LEVEL", "debug")
	os.Setenv("SW_APM_COLLECTOR", addr)
	os.Setenv("SW_APM_TRUSTEDPATH", testCertFile)

	// set proxy
	os.Setenv("SW_APM_PROXY", proxyUrl)
	os.Setenv("SW_APM_PROXY_CERT_PATH", testCertFile)
	proxy, err := NewTestProxyServer(proxyUrl, testCertFile, testKeyFile)
	require.Nil(t, err)
	require.Nil(t, proxy.Start())
	defer proxy.Stop()

	config.Load()

	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	oldReporter := globalReporter
	defer func() { globalReporter = oldReporter }()

	setGlobalReporter("ssl")

	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)

	// Test WaitForReady
	// The reporter is not ready when there is no default setting.
	ctxTm1, cancel1 := context.WithTimeout(context.Background(), 0)
	defer cancel1()
	assert.False(t, r.WaitForReady(ctxTm1))

	// The reporter becomes ready after it has got the default setting.
	ready := make(chan bool, 1)
	r.getSettings(ready)
	ctxTm2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel2()
	assert.True(t, r.WaitForReady(ctxTm2))
	assert.True(t, r.isReady())

	ctx := newTestContext(t)
	ev1, err := ctx.newEvent(LabelInfo, "layer1")
	assert.NoError(t, err)
	ev2, err := ctx.newEvent(LabelInfo, "layer2")
	assert.NoError(t, err)

	assert.Error(t, r.reportEvent(nil, nil))
	assert.Error(t, r.reportEvent(ctx, nil))
	assert.NoError(t, r.reportEvent(ctx, ev1))

	assert.Error(t, r.reportStatus(nil, nil))
	assert.Error(t, r.reportStatus(ctx, nil))
	// time.Sleep(time.Second)
	assert.NoError(t, r.reportStatus(ctx, ev2))

	assert.Equal(t, addr, r.conn.address)

	assert.Equal(t, TestServiceKey, r.serviceKey.Load())

	assert.Equal(t, int32(grpcMetricIntervalDefault), r.collectMetricInterval)
	assert.Equal(t, grpcGetSettingsIntervalDefault, r.getSettingsInterval)
	assert.Equal(t, grpcSettingsTimeoutCheckIntervalDefault, r.settingsTimeoutCheckInterval)

	time.Sleep(time.Second)

	// The reporter becomes not ready after the default setting has been deleted
	removeSetting("")
	r.checkSettingsTimeout(make(chan bool, 1))

	assert.False(t, r.isReady())
	ctxTm3, cancel3 := context.WithTimeout(context.Background(), 0)
	assert.False(t, r.WaitForReady(ctxTm3))
	defer cancel3()

	// stop test reporter
	server.Stop()

	// assert data received
	require.Len(t, server.events, 1)
	assert.Equal(t, server.events[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.events[0].Messages, 1)

	require.Len(t, server.status, 1)
	assert.Equal(t, server.status[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.status[0].Messages, 1)

	dec1, dec2 := mbson.M{}, mbson.M{}
	err = mbson.Unmarshal(server.events[0].Messages[0], &dec1)
	require.NoError(t, err)
	err = mbson.Unmarshal(server.status[0].Messages[0], &dec2)
	require.NoError(t, err)

	assert.Equal(t, dec1["Layer"], "layer1")
	assert.Equal(t, dec1["Hostname"], host.Hostname())
	assert.Equal(t, dec1["Label"], LabelInfo)
	assert.Equal(t, dec1["PID"], host.PID())

	assert.Equal(t, dec2["Layer"], "layer2")
}

func TestHttpProxy(t *testing.T) {
	testProxy(t, "http://usr:pwd@localhost:12345")
}

func TestHttpsProxy(t *testing.T) {
	testProxy(t, "https://usr:pwd@localhost:12345")
}

func TestFlush(t *testing.T) {
	globalReporter = newGRPCReporter()
	r := globalReporter.(*grpcReporter)
	assert.NoError(t, r.Flush())
}