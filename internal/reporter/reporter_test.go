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
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/reporter/mocks"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/atomic"
	"io"
	stdlog "log"
	"os"
	"testing"
	"time"

	"strings"

	pb "github.com/solarwindscloud/apm-proto/go/collectorpb"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mbson "gopkg.in/mgo.v2/bson"
)

const TestServiceKey = "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go"

func setEnv(k string, v string) {
	if err := os.Setenv(k, v); err != nil {
		stdlog.Panic("could not set env!", err)
	}
}

// this runs before init()
var _ = func() (_ struct{}) {
	periodicTasksDisabled = true

	os.Clearenv()
	setEnv("SW_APM_SERVICE_KEY", TestServiceKey)
	setEnv("SW_APM_DEBUG_LEVEL", "debug")

	config.Load()
	return
}()

// ========================= NULL Reporter =============================

func TestNullReporter(t *testing.T) {
	nullR := &nullReporter{}
	require.NoError(t, nullR.ReportEvent(nil))
	require.NoError(t, nullR.ReportStatus(nil))
}

// ========================= GRPC Reporter =============================

func TestGRPCReporter(t *testing.T) {
	// start test gRPC server
	setEnv("SW_APM_DEBUG_LEVEL", "debug")
	config.Load()
	addr := "localhost:4567"
	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	// set gRPC reporter
	setEnv("SW_APM_COLLECTOR", addr)
	setEnv("SW_APM_TRUSTEDPATH", testCertFile)
	config.Load()
	oldReporter := globalReporter
	setGlobalReporter("ssl", "")

	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)

	// Test WaitForReady
	// The reporter is not ready when there is no default setting.
	ctxTm1, cancel1 := context.WithTimeout(context.Background(), 0)
	defer cancel1()
	require.False(t, r.WaitForReady(ctxTm1))

	// The reporter becomes ready after it has got the default setting.
	ready := make(chan bool, 1)
	r.getSettings(ready)
	ctxTm2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel2()
	require.True(t, r.WaitForReady(ctxTm2))
	require.True(t, r.isReady())

	ev1 := CreateInfoEvent(validSpanContext, time.Now())
	ev1.SetLayer("layer1")
	ev2 := CreateInfoEvent(validSpanContext, time.Now())
	ev2.SetLayer("layer2")

	require.Error(t, r.ReportEvent(nil))
	require.Error(t, r.ReportEvent(nil))
	require.NoError(t, r.ReportEvent(ev1))

	require.Error(t, r.ReportStatus(nil))
	require.Error(t, r.ReportStatus(nil))
	require.NoError(t, r.ReportStatus(ev2))

	require.Equal(t, addr, r.conn.address)

	require.Equal(t, TestServiceKey, r.serviceKey.Load())

	require.Equal(t, int32(metrics.ReportingIntervalDefault), r.collectMetricInterval)
	require.Equal(t, grpcGetSettingsIntervalDefault, r.getSettingsInterval)
	require.Equal(t, grpcSettingsTimeoutCheckIntervalDefault, r.settingsTimeoutCheckInterval)

	time.Sleep(time.Second)

	// The reporter becomes not ready after the default setting has been deleted
	removeSetting()
	r.checkSettingsTimeout(make(chan bool, 1))

	require.False(t, r.isReady())
	ctxTm3, cancel3 := context.WithTimeout(context.Background(), 0)
	require.False(t, r.WaitForReady(ctxTm3))
	defer cancel3()

	// stop test reporter
	server.Stop()
	globalReporter = oldReporter

	// assert data received
	require.Len(t, server.events, 1)
	require.Equal(t, server.events[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.events[0].Messages, 1)

	require.Len(t, server.status, 1)
	require.Equal(t, server.status[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.status[0].Messages, 1)

	dec1, dec2 := mbson.M{}, mbson.M{}
	err := mbson.Unmarshal(server.events[0].Messages[0], &dec1)
	require.NoError(t, err)
	err = mbson.Unmarshal(server.status[0].Messages[0], &dec2)
	require.NoError(t, err)

	require.Equal(t, dec1["Layer"], "layer1")
	require.Equal(t, dec1["Hostname"], host.Hostname())
	require.Equal(t, dec1["Label"], constants.InfoLabel)
	require.Equal(t, dec1["PID"], host.PID())

	require.Equal(t, dec2["Layer"], "layer2")
}

func TestShutdownGRPCReporter(t *testing.T) {
	// start test gRPC server
	setEnv("SW_APM_DEBUG_LEVEL", "debug")
	addr := "localhost:4567"
	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	// set gRPC reporter
	setEnv("SW_APM_COLLECTOR", addr)
	setEnv("SW_APM_TRUSTEDPATH", testCertFile)
	config.Load()
	oldReporter := globalReporter
	setGlobalReporter("ssl", "")

	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)
	r.ShutdownNow()

	require.Equal(t, true, r.Closed())

	r.ShutdownNow()

	// stop test reporter
	server.Stop()
	globalReporter = oldReporter
}

func TestSetServiceKey(t *testing.T) {
	r := &grpcReporter{serviceKey: atomic.NewString("unset")}
	err := r.SetServiceKey("foo")
	require.Error(t, err)
	require.Equal(t, "invalid service key format", err.Error())

	err = r.SetServiceKey(TestServiceKey)
	require.NoError(t, err)
	require.Equal(t, TestServiceKey, r.serviceKey.Load())
}

func TestGetServiceName(t *testing.T) {
	r := &grpcReporter{serviceKey: atomic.NewString(TestServiceKey), otelServiceName: ""}
	require.Equal(t, "go", r.GetServiceName())
	r = &grpcReporter{serviceKey: atomic.NewString(TestServiceKey), otelServiceName: "override"}
	require.Equal(t, "override", r.GetServiceName())
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
	setEnv("SW_APM_DEBUG_LEVEL", "debug")
	oldKey := os.Getenv("SW_APM_SERVICE_KEY")
	setEnv("SW_APM_SERVICE_KEY", invalidKey)
	addr := "localhost:4567"
	setEnv("SW_APM_COLLECTOR", addr)
	setEnv("SW_APM_TRUSTEDPATH", testCertFile)

	// start test gRPC server
	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	// set gRPC reporter
	config.Load()
	oldReporter := globalReporter

	log.SetLevel(log.INFO)
	setGlobalReporter("ssl", "")
	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)
	ev1 := CreateInfoEvent(validSpanContext, time.Now())
	ev1.SetLayer("hello-from-invalid-key")
	require.NoError(t, r.ReportEvent(ev1))

	time.Sleep(time.Second)

	// The agent reporter should be closed due to received INVALID_API_KEY from the collector
	require.Equal(t, true, r.Closed())

	r.ShutdownNow()

	// Tear down everything.
	server.Stop()
	globalReporter = oldReporter
	setEnv("SW_APM_SERVICE_KEY", oldKey)

	patterns := []string{
		"rsp=INVALID_API_KEY",
		"Shutting down the reporter",
		"eventSender goroutine exiting",
		"statusSender goroutine exiting",
		"eventBatchSender goroutine exiting",
	}
	for _, ptn := range patterns {
		require.True(t, strings.Contains(buf.String(), ptn), buf.String()+"^^^^^^"+ptn)
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
		_ = DefaultBackoff(i, bf)
	}
	require.Equal(t, expected, backoff)
	require.NotNil(t, DefaultBackoff(grpcMaxRetries+1, func(d time.Duration) {}))
}

type NoopDialer struct{}

func (d *NoopDialer) Dial(DialParams) (*grpc.ClientConn, error) {
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
	require.Equal(t, errReporterExiting, c.InvokeRPC(exit, mockMethod))

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

	require.Equal(t, errInvalidServiceKey, c.InvokeRPC(exit, mockMethod))

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
	require.Equal(t, errNoRetryOnErr, c.InvokeRPC(exit, mockMethod))

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
	require.Equal(t, nil, c.InvokeRPC(exit, mockMethod))
	require.True(t, strings.Contains(buf.String(), "invocation error"))
	require.True(t, strings.Contains(buf.String(), "error recovered"))

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
	require.Equal(t, nil, c.InvokeRPC(exit, mockMethod))
	require.True(t, c.isActive())
	require.Equal(t, "new-addr:9999", c.address)

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

	require.Contains(t, c.InvokeRPC(exit, mockMethod).Error(), errNoRetryOnErr.Error())
}

func TestInitReporter(t *testing.T) {
	// Test disable agent
	setEnv("SW_APM_ENABLED", "false")
	config.Load()
	initReporter(resource.Empty())
	require.IsType(t, &nullReporter{}, globalReporter)

	// Test enable agent
	require.NoError(t, os.Unsetenv("SW_APM_ENABLED"))
	setEnv("SW_APM_REPORTER", "ssl")
	config.Load()
	require.True(t, config.GetEnabled())

	initReporter(resource.NewWithAttributes("", semconv.ServiceName("my service name")))
	require.IsType(t, &grpcReporter{}, globalReporter)
	require.Equal(t, "my service name", globalReporter.GetServiceName())
}

func TestCollectMetricsNextInterval(t *testing.T) {
	r := &grpcReporter{collectMetricInterval: 10}
	next := r.collectMetricsNextInterval()
	// very weak check
	require.True(t, next <= time.Second*10, next)
}

// testProxy performs tests of http/https proxy.
func testProxy(t *testing.T, proxyUrl string) {
	addr := "localhost:4567"

	setEnv("SW_APM_DEBUG_LEVEL", "debug")
	setEnv("SW_APM_COLLECTOR", addr)
	setEnv("SW_APM_TRUSTEDPATH", testCertFile)

	// set proxy
	setEnv("SW_APM_PROXY", proxyUrl)
	setEnv("SW_APM_PROXY_CERT_PATH", testCertFile)
	proxy, err := NewTestProxyServer(proxyUrl, testCertFile, testKeyFile)
	require.Nil(t, err)
	require.Nil(t, proxy.Start())
	defer func() {
		require.NoError(t, proxy.Stop())
	}()

	config.Load()

	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)

	oldReporter := globalReporter
	defer func() { globalReporter = oldReporter }()

	setGlobalReporter("ssl", "")

	require.IsType(t, &grpcReporter{}, globalReporter)

	r := globalReporter.(*grpcReporter)

	// Test WaitForReady
	// The reporter is not ready when there is no default setting.
	ctxTm1, cancel1 := context.WithTimeout(context.Background(), 0)
	defer cancel1()
	require.False(t, r.WaitForReady(ctxTm1))

	// The reporter becomes ready after it has got the default setting.
	ready := make(chan bool, 1)
	r.getSettings(ready)
	ctxTm2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel2()
	require.True(t, r.WaitForReady(ctxTm2))
	require.True(t, r.isReady())

	ev1 := CreateInfoEvent(validSpanContext, time.Now())
	ev1.SetLayer("layer1")
	require.NoError(t, err)
	ev2 := CreateInfoEvent(validSpanContext, time.Now())
	ev2.SetLayer("layer2")
	require.NoError(t, err)

	require.Error(t, r.ReportEvent(nil))
	require.Error(t, r.ReportEvent(nil))
	require.NoError(t, r.ReportEvent(ev1))

	require.Error(t, r.ReportStatus(nil))
	require.Error(t, r.ReportStatus(nil))
	// time.Sleep(time.Second)
	require.NoError(t, r.ReportStatus(ev2))

	require.Equal(t, addr, r.conn.address)

	require.Equal(t, TestServiceKey, r.serviceKey.Load())

	require.Equal(t, int32(metrics.ReportingIntervalDefault), r.collectMetricInterval)
	require.Equal(t, grpcGetSettingsIntervalDefault, r.getSettingsInterval)
	require.Equal(t, grpcSettingsTimeoutCheckIntervalDefault, r.settingsTimeoutCheckInterval)

	time.Sleep(time.Second)

	// The reporter becomes not ready after the default setting has been deleted
	removeSetting()
	r.checkSettingsTimeout(make(chan bool, 1))

	require.False(t, r.isReady())
	ctxTm3, cancel3 := context.WithTimeout(context.Background(), 0)
	require.False(t, r.WaitForReady(ctxTm3))
	defer cancel3()

	// stop test reporter
	server.Stop()

	// assert data received
	require.Len(t, server.events, 1)
	require.Equal(t, server.events[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.events[0].Messages, 1)

	require.Len(t, server.status, 1)
	require.Equal(t, server.status[0].Encoding, pb.EncodingType_BSON)
	require.Len(t, server.status[0].Messages, 1)

	dec1, dec2 := mbson.M{}, mbson.M{}
	err = mbson.Unmarshal(server.events[0].Messages[0], &dec1)
	require.NoError(t, err)
	err = mbson.Unmarshal(server.status[0].Messages[0], &dec2)
	require.NoError(t, err)

	require.Equal(t, dec1["Layer"], "layer1")
	require.Equal(t, dec1["Hostname"], host.Hostname())
	require.Equal(t, dec1["Label"], constants.InfoLabel)
	require.Equal(t, dec1["PID"], host.PID())

	require.Equal(t, dec2["Layer"], "layer2")
}

func TestHttpProxy(t *testing.T) {
	testProxy(t, "http://usr:pwd@localhost:12345")
}

func TestHttpsProxy(t *testing.T) {
	testProxy(t, "https://usr:pwd@localhost:12345")
}
