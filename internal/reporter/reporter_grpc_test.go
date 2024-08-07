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

// There's an issue in GitHub Actions running Windows that causes `net.Listen`
// to fail with:
//
// 		listen tcp: lookup localhost: getaddrinfow: A non-recoverable error
//		occurred during a database lookup.
//
// For now, we're going to rely on non-windows to verify that this code works.
// Over time, the (non-OTLP) grpc implementation will be phased out, so I think
// this is a reasonable compromise.
// -- @swi-jared

//go:build !windows

package reporter

import (
	"crypto/subtle"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/solarwinds/apm-go/internal/utils"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"context"
	pb "github.com/solarwinds/apm-proto/go/collectorpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	testKeyFile  = path.Join(".", "for_test.key")
	testCertFile = path.Join(".", "for_test.crt")
)

type TestGRPCServer struct {
	t          *testing.T
	grpcServer *grpc.Server
	addr       string
	// The mutex to protect the other fields, mainly the slices below as gRPC needs concurrency-safe
	// Performance is not a concern for a testing reporter, so we are fine with a single mutex for all
	// the fields.
	mutex   sync.Mutex
	events  []*pb.MessageRequest
	metrics []*pb.MessageRequest
	status  []*pb.MessageRequest
	pings   int
}

func StartTestGRPCServer(t *testing.T, addr string) *TestGRPCServer {
	lis, err := net.Listen("tcp", addr)
	require.NoError(t, err)

	// Create the TLS credentials
	creds, err := credentials.NewServerTLSFromFile(testCertFile, testKeyFile)
	require.NoError(t, err, "could not load TLS keys")
	assert.NotNil(t, creds)

	// Create the gRPC server with the credentials
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	assert.NotNil(t, grpcServer)
	testServer := &TestGRPCServer{t: t, grpcServer: grpcServer, addr: addr}
	pb.RegisterTraceCollectorServer(grpcServer, testServer)
	require.NoError(t, err)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	return testServer
}

func printMessageRequest(req *pb.MessageRequest) {
	bs, _ := json.Marshal(req)
	fmt.Printf("Raw message marshaled to json->%s\n", bs)
	fmt.Println("Events decoded from BSON->")
	for idx, m := range req.Messages {
		fmt.Printf("#%d->", idx)
		fmt.Println(utils.SPrintBson(m))
	}
}

func printSettingsRequest(req *pb.SettingsRequest) {
	bs, _ := json.Marshal(req)
	fmt.Printf("Raw message marshaled to json->%s\n", bs)
}

func (s *TestGRPCServer) Stop() { s.grpcServer.Stop() }

func (s *TestGRPCServer) PostEvents(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	fmt.Println("TestGRPCServer.PostEvents req:")
	printMessageRequest(req)
	s.events = append(s.events, req)
	if strings.HasPrefix(req.ApiKey, "invalid") {
		return &pb.MessageResult{Result: pb.ResultCode_INVALID_API_KEY}, nil
	}
	return &pb.MessageResult{Result: pb.ResultCode_OK}, nil
}

func (s *TestGRPCServer) PostMetrics(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	fmt.Println("TestGRPCServer.PostMetrics req:")
	printMessageRequest(req)
	s.metrics = append(s.metrics, req)
	return &pb.MessageResult{Result: pb.ResultCode_OK}, nil
}

func (s *TestGRPCServer) PostStatus(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	fmt.Println("TestGRPCServer.PostStatus req:")
	printMessageRequest(req)
	s.status = append(s.status, req)
	return &pb.MessageResult{Result: pb.ResultCode_OK}, nil
}

func (s *TestGRPCServer) GetSettings(ctx context.Context, req *pb.SettingsRequest) (*pb.SettingsResult, error) {
	fmt.Println("TestGRPCServer.GetSettings req:")
	printSettingsRequest(req)
	return &pb.SettingsResult{
		Result: pb.ResultCode_OK,
		Settings: []*pb.OboeSetting{{
			Type: pb.OboeSettingType_DEFAULT_SAMPLE_RATE,
			// Flags:     XXX,
			// Timestamp: XXX,
			Value:     1000000,
			Arguments: map[string][]byte{
				//   "BucketCapacity": XXX,
				//   "BucketRate":     XXX,
			},
			Ttl: 120,
		}},
	}, nil
}

func (s *TestGRPCServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.MessageResult, error) {
	fmt.Printf("TestGRPCServer.Ping with APIKey: %s\n", req.ApiKey)
	s.pings++
	return &pb.MessageResult{Result: pb.ResultCode_OK}, nil
}

// TestProxyServer is a simple proxy server prototype which supports http, https and socks5.
// It's not production-ready and should only be used for tests.
type TestProxyServer struct {
	url       *url.URL
	pemFile   string
	keyFile   string
	closeFunc func() error
}

func NewTestProxyServer(rawUrl string, pem, key string) (*TestProxyServer, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create test proxy server: %w", err)
	}
	return &TestProxyServer{url: u, pemFile: pem, keyFile: key}, nil
}

func (p *TestProxyServer) Start() error {
	srv := &http.Server{Addr: p.url.Host, Handler: http.HandlerFunc(p.proxyHttpHandler)}

	closeFunc := func() error {
		return srv.Close()
	}
	switch p.url.Scheme {
	case "http":
		go func() {
			_ = srv.ListenAndServe()
		}()
		p.closeFunc = closeFunc
	case "https":
		go func() {
			_ = srv.ListenAndServeTLS(p.pemFile, p.keyFile)
		}()
		p.closeFunc = closeFunc
	// TODO: case "socks5":
	default:
		panic(fmt.Sprintf("Unsupported proxy type: %s", p.url.Scheme))
	}

	return nil
}

func (p *TestProxyServer) Stop() error {
	if p.closeFunc != nil {
		return p.closeFunc()
	}
	return errors.New("no close function found")
}

// Ref: https://medium.com/@mlowicki/http-s-proxy-in-golang-in-less-than-100-lines-of-code-6a51c2f2c38c
func (p *TestProxyServer) proxyHttpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		panic("CONNECT only, please")
	}

	r.Header.Add("Authorization", r.Header.Get("Proxy-Authorization")) // a dirty hack
	user, pwd, ok := r.BasicAuth()
	expectedUser := p.url.User.Username()
	expectedPwd, _ := p.url.User.Password()

	if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(expectedUser)) != 1 ||
		subtle.ConstantTimeCompare([]byte(pwd), []byte(expectedPwd)) != 1 {
		w.Header().Set("WWW-Authenticate", `Basic realm="wrong auth"`)
		w.WriteHeader(401)
		if _, err := w.Write([]byte("Unauthorised.\n")); err != nil {
			panic(err)
		}
		return
	}

	serverConn, err := net.DialTimeout("tcp", r.Host, 1*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking failed", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = clientConn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	if err != nil {
		panic(err)
	}

	go forward(serverConn, clientConn)
	go forward(clientConn, serverConn)
}

func forward(dst io.WriteCloser, src io.ReadCloser) {
	defer func() {
		_ = dst.Close()
		_ = src.Close()
	}()
	_, _ = io.Copy(dst, src)
}

func TestAppopticsCertificate(t *testing.T) {
	certPool := x509.NewCertPool()
	cert := []byte(legacyAOcertificate)
	require.True(t, certPool.AppendCertsFromPEM(cert))
}
