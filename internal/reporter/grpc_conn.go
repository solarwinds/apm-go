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

package reporter

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/utils"
	collector "github.com/solarwinds/apm-proto/go/collectorpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
)

const (
	legacyAOcertificate = `-----BEGIN CERTIFICATE-----
MIID8TCCAtmgAwIBAgIJAMoDz7Npas2/MA0GCSqGSIb3DQEBCwUAMIGOMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5j
aXNjbzEVMBMGA1UECgwMTGlicmF0byBJbmMuMRUwEwYDVQQDDAxBcHBPcHRpY3Mg
Q0ExJDAiBgkqhkiG9w0BCQEWFXN1cHBvcnRAYXBwb3B0aWNzLmNvbTAeFw0xNzA5
MTUyMjAxMzlaFw0yNzA5MTMyMjAxMzlaMIGOMQswCQYDVQQGEwJVUzETMBEGA1UE
CAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzEVMBMGA1UECgwM
TGlicmF0byBJbmMuMRUwEwYDVQQDDAxBcHBPcHRpY3MgQ0ExJDAiBgkqhkiG9w0B
CQEWFXN1cHBvcnRAYXBwb3B0aWNzLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAOxO0wsGba3iI4r3L5BMST0rAO/gGaUhpQre6nRwVTmPCnLw1bmn
GdiFgYv/oRRwU+VieumHSQqoOmyFrg+ajGmvUDp2WqQ0It+XhcbaHFiAp2H7+mLf
cUH6S43/em0WUxZHeRzRupRDyO1bX6Hh2jgxykivlFrn5HCIQD5Hx1/SaZoW9v2n
oATCbgFOiPW6kU/AVs4R0VBujon13HCehVelNKkazrAEBT1i6RvdOB6aQQ32seW+
gLV5yVWSPEJvA9ZJqad/nQ8EQUMSSlVN191WOjp4bGpkJE1svs7NmM+Oja50W56l
qOH5eWermr/8qWjdPlDJ+I0VkgN0UyHVuRECAwEAAaNQME4wHQYDVR0OBBYEFOuL
KDTFhRQXwlBRxhPqhukrNYeRMB8GA1UdIwQYMBaAFOuLKDTFhRQXwlBRxhPqhukr
NYeRMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAJQtH446NZhjusy6
iCyvmnD95ybfNPDpjHmNx5n9Y6w9n+9y1o3732HUJE+WjvbLS3h1o7wujGKMcRJn
7I7eTDd26ZhLvnh5/AitYjdxrtUkQDgyxwLFJKhZu0ik2vXqj0fL961/quJL8Gyp
hNj3Nf7WMohQMSohEmCCX2sHyZGVGYmQHs5omAtkH/NNySqmsWNcpgd3M0aPDRBZ
5VFreOSGKBTJnoLNqods/S9RV0by84hm3j6aQ/tMDIVE9VCJtrE6evzC0MWyVFwR
ftgwcxyEq5SkiR+6BCwdzAMqADV37TzXDHLjwSrMIrgLV5xZM20Kk6chxI5QAr/f
7tsqAxw=
-----END CERTIFICATE-----`

	grpcRetryDelayMax = 60 // max connection/send retry delay in seconds
	grpcMaxRetries    = 20 // The message will be dropped after this number of retries
)

// everything needed for a GRPC connection
type grpcConnection struct {
	name           string                         // connection name
	client         collector.TraceCollectorClient // GRPC client instance
	connection     *grpc.ClientConn               // GRPC connection object
	address        string                         // collector address
	certificate    string                         // collector certificate
	pingTicker     *time.Timer                    // timer for keep alive pings in seconds
	pingTickerLock sync.Mutex                     // lock to ensure sequential access of pingTicker
	lock           sync.RWMutex                   // lock to ensure sequential access (in case of connection loss)
	queueStats     *metrics.EventQueueStats       // queue stats (reset on each metrics report cycle)

	proxy            string
	proxyTLSCertPath string

	// atomicActive indicates if the underlying connection is active. It should
	// be reconnected or redirected to a new address in case of inactive. The
	// value 0 represents false and a value other than 0 (usually 1) means true
	atomicActive int32

	// the backoff function
	backoff Backoff
	Dialer
	maxReqBytes int64 // the maximum size for an RPC request body
	refCount    int   // counts clients of this grpc connection
}

type Backoff func(retries int, wait func(d time.Duration)) error

// DefaultBackoff calls the wait function to sleep for a certain time based on
// the retries value. It returns immediately if the retries exceeds a threshold.
func DefaultBackoff(retries int, wait func(d time.Duration)) error {
	if retries > grpcMaxRetries {
		return errGiveUpAfterRetries
	}
	delay := int(grpcRetryDelayInitial * math.Pow(grpcRetryDelayMultiplier, float64(retries-1)))
	if delay > grpcRetryDelayMax*1000 {
		delay = grpcRetryDelayMax * 1000
	}

	wait(time.Duration(delay) * time.Millisecond)
	return nil
}

// GrpcConnOpt defines the function type that sets an option of the grpcConnection
type GrpcConnOpt func(c *grpcConnection)

// WithCert returns a function that sets the certificate
func WithCert(cert string) GrpcConnOpt {
	return func(c *grpcConnection) {
		c.certificate = cert
	}
}

// WithProxy assign the proxy url to the gRPC connection
func WithProxy(proxy string) GrpcConnOpt {
	return func(c *grpcConnection) {
		c.proxy = proxy
	}
}

// WithProxyCertPath assigns the proxy TLS certificate path to the gRPC connection
func WithProxyCertPath(certPath string) GrpcConnOpt {
	return func(c *grpcConnection) {
		c.proxyTLSCertPath = certPath
	}
}

// WithMaxReqBytes sets the maximum size of an RPC request
func WithMaxReqBytes(size int64) GrpcConnOpt {
	return func(c *grpcConnection) {
		c.maxReqBytes = size
	}
}

// WithDialer returns a function that sets the Dialer option
func WithDialer(d Dialer) GrpcConnOpt {
	return func(c *grpcConnection) {
		c.Dialer = d
	}
}

// WithBackoff return a function that sets the backoff option
func WithBackoff(b Backoff) GrpcConnOpt {
	return func(c *grpcConnection) {
		c.backoff = b
	}
}

func newGrpcConnection(name string, target string, opts ...GrpcConnOpt) (*grpcConnection, error) {
	gc := &grpcConnection{
		name:        name,
		client:      nil,
		connection:  nil,
		address:     target,
		certificate: "",
		pingTicker:  time.NewTimer(time.Duration(grpcPingIntervalDefault) * time.Second),
		queueStats:  &metrics.EventQueueStats{},
		backoff:     DefaultBackoff,
		Dialer:      &DefaultDialer{},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(gc)
		}
	}

	err := gc.connect()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to connect to %s", name), err)
	}
	return gc, nil
}

func (c *grpcConnection) AddClient() {
	c.lock.Lock()
	c.refCount++
	defer c.lock.Unlock()
}

// Close closes the gRPC connection and set the pointer to nil
func (c *grpcConnection) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.refCount--

	if c.refCount == 0 && c.connection != nil {
		if err := c.connection.Close(); err != nil {
			log.Warning("error when closing connection; ignoring", err)
		}
		c.connection = nil
	}
}

func (c *grpcConnection) setAddress(addr string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.address = addr
	c.setActive(false)
}

// connect does the operation of connecting to a collector. It may be the same
// address or a new one. Those who issue the connection request need to set
// the stale flag to true, otherwise this function will do nothing.
func (c *grpcConnection) connect() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Skip it if the connection is not stale - someone else may have done
	// the connection.
	if c.isActive() {
		log.Debugf("[%s] Someone else has done the redirection.", c.name)
		return nil
	}
	// create a new connection object for this client
	conn, err := c.Dial(DialParams{
		Certificate:   c.certificate,
		Address:       c.address,
		Proxy:         c.proxy,
		ProxyCertPath: c.proxyTLSCertPath,
	})
	if err != nil {
		return errors.Join(fmt.Errorf("failed to connect to %s", c.address), err)
	}

	// close the old connection
	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			log.Warning("error when closing connection; ignoring", err)
		}
	}
	// set new connection (need to be protected)
	c.connection = conn
	c.client = collector.NewTraceCollectorClient(c.connection)
	c.setActive(true)

	log.Infof("[%s] Connected to %s", c.name, c.address)
	return nil
}

func (c *grpcConnection) isActive() bool {
	return atomic.LoadInt32(&c.atomicActive) == 1
}

func (c *grpcConnection) setActive(active bool) {
	var flag int32
	if active {
		flag = 1
	}
	atomic.StoreInt32(&c.atomicActive, flag)
}

func (c *grpcConnection) reconnect() error {
	return c.connect()
}

// ========================= Ping Handling =============================

// reset keep alive timer on a given GRPC connection
func (c *grpcConnection) resetPing() {
	if c.pingTicker == nil {
		return
	}
	c.pingTickerLock.Lock()
	// TODO: Reset may run into a race condition
	c.pingTicker.Reset(time.Duration(grpcPingIntervalDefault) * time.Second)
	c.pingTickerLock.Unlock()
}

// send a keep alive (ping) request on a given GRPC connection
func (c *grpcConnection) ping(exit chan struct{}, key string) error {
	method := newPingMethod(key, c.name)
	err := c.InvokeRPC(exit, method)
	log.Debug(method.CallSummary())
	return err
}

// possible errors while issuing an RPC call
var (
	// The collector notifies that the service key of this reporter is invalid.
	// The reporter should be closed in this case.
	errInvalidServiceKey = errors.New("invalid service key")

	// Only a certain amount of retries are allowed. The message will be dropped
	// if the number of retries exceeds this number.
	errGiveUpAfterRetries = errors.New("give up after retries")

	// The maximum number of redirections has reached and the message will be
	// dropped.
	errTooManyRedirections = errors.New("too many redirections")

	// the operation or loop cannot continue as the reporter is exiting.
	errReporterExiting = errors.New("reporter is exiting")

	// errNoRetryOnErr means this RPC call method doesn't need retry, e.g., the
	// Ping method.
	errNoRetryOnErr = errors.New("method requires no retry")

	// errConnStale means the connection is broken. This usually happens
	// when an RPC call is timeout.
	errConnStale     = errors.New("connection is stale")
	errRequestTooBig = errors.New("RPC request is too big")
)

// InvokeRPC makes an RPC call and returns an error if something is broken and
// cannot be handled by itself, e.g., the collector's response indicates the
// service key is invalid. It maintains the connection and does the retries
// automatically and transparently. It may give up after a certain times of
// retries, so it is a best-effort service only.
//
// When an error is returned, it usually means a fatal error and the reporter
// may be shutdown.
func (c *grpcConnection) InvokeRPC(exit <-chan struct{}, m Method) error {
	c.queueStats.SetQueueLargest(m.MessageLen())

	// counter for redirects so we know when the limit has been reached
	redirects := 0
	// Number of gRPC errors encountered
	failsNum := 0
	// Number of retries, including gRPC errors and collector errors
	retriesNum := 0

	printRPCMsg(m)

	for {
		// Fail-fast in case the reporter has been closed, avoid retrying in
		// this case.
		select {
		case <-exit:
		default:
		}
		var err = errConnStale
		// Protect the call to the client object or we could run into problems
		// if another goroutine is messing with it at the same time, e.g. doing
		// a redirection.
		c.lock.RLock()
		if c.isActive() {
			ctx, cancel := context.WithTimeout(context.Background(), grpcCtxTimeout)
			if m.RequestSize() > c.maxReqBytes {
				err = fmt.Errorf("rpc request exceeds byte limit; request size: %d, max size: %d", m.RequestSize(), c.maxReqBytes)
			} else {
				if m.ServiceKey() != "" {
					err = m.Call(ctx, c.client)
				} else {
					err = nil
					log.Infof("%s is skipped as no service key is assigned.", m.String())
				}
			}

			code := status.Code(err)
			if code == codes.DeadlineExceeded ||
				code == codes.Canceled {
				log.Infof("[%s] Connection becomes stale: %v.", c.name, err)
				err = errConnStale
				c.setActive(false)
			}
			cancel()
		}
		c.lock.RUnlock()

		// we sent something, or at least tried to, so we're not idle - reset
		// the keepalive timer
		c.resetPing()

		if err != nil {
			// gRPC handles the reconnection automatically.
			failsNum++
			if failsNum == grpcRetryLogThreshold {
				log.Warningf("[%s] invocation error: %v.", m, err)
			} else {
				log.Debugf("[%s] (%v) invocation error: %v.", m, failsNum, err)
			}
		} else {
			if failsNum >= grpcRetryLogThreshold {
				log.Warningf("[%s] error recovered.", m)
			}
			failsNum = 0

			// server responded, check the result code and perform actions accordingly
			switch result, _ := m.ResultCode(); result {
			case collector.ResultCode_OK:
				c.queueStats.NumSentAdd(m.MessageLen())
				return nil

			case collector.ResultCode_TRY_LATER:
				log.Info(m.CallSummary())
				c.queueStats.NumFailedAdd(m.MessageLen())
			case collector.ResultCode_LIMIT_EXCEEDED:
				log.Info(m.CallSummary())
				c.queueStats.NumFailedAdd(m.MessageLen())
			case collector.ResultCode_INVALID_API_KEY:
				log.Error(m.CallSummary())
				return errInvalidServiceKey
			case collector.ResultCode_REDIRECT:
				log.Warning(m.CallSummary())
				redirects++

				if redirects > grpcRedirectMax {
					return errTooManyRedirections
				} else if m.Arg() != "" {
					c.setAddress(m.Arg())
					// a proper redirect shouldn't cause delays
					retriesNum = 0
				} else {
					log.Warning(fmt.Errorf("redirection target is empty for %s", c.name))
				}
			default:
				log.Info(m.CallSummary())
			}
		}

		if !c.isActive() {
			if err = c.reconnect(); err != nil {
				return err
			}
		}

		if !m.RetryOnErr(err) {
			if err != nil {
				return errors.Join(errNoRetryOnErr, err)
			} else {
				return errNoRetryOnErr
			}
		}

		retriesNum++
		err = c.backoff(retriesNum, func(d time.Duration) {
			time.Sleep(d)
		})
		if err != nil {
			return err
		}
	}
}

// Dialer has a method Dial which accepts a grpcConnection object as the
// argument and returns a ClientConn object.
type Dialer interface {
	Dial(options DialParams) (*grpc.ClientConn, error)
}

type DialParams struct {
	Certificate   string
	Address       string
	Proxy         string
	ProxyCertPath string
}

// DefaultDialer implements the Dialer interface to provide the default dialing
// method.
type DefaultDialer struct{}

// Dial issues the connection to the remote address with attributes provided by
// the grpcConnection.
func (d *DefaultDialer) Dial(p DialParams) (*grpc.ClientConn, error) {
	var certPool *x509.CertPool
	var err error

	// If the certificate is not overriden and the address contains
	// `appoptics.com`, then we need to specify the self-signed certificate. The
	// original code did this by default since AO had a self-signed certificate,
	// whereas SolarWinds Observability has a CA-signed cert.
	if p.Certificate == "" && strings.Contains(p.Address, "appoptics.com") {
		log.Debug("Defaulting to Appoptics certificate")
		p.Certificate = legacyAOcertificate
	}

	if p.Certificate != "" {
		certPool = x509.NewCertPool()
		cert := []byte(p.Certificate)
		if ok := certPool.AppendCertsFromPEM(cert); !ok {
			return nil, errors.New("unable to append the certificate to pool")
		}
	} else {
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, errors.Join(errors.New("unable to obtain system cert pool"), err)
		}
	}

	// trim port from server name used for TLS verification
	serverName := p.Address
	if s := strings.Split(p.Address, ":"); len(s) > 0 {
		serverName = s[0]
	}

	tlsConfig := &tls.Config{
		ServerName: serverName,
		RootCAs:    certPool,
	}

	creds := credentials.NewTLS(tlsConfig)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
	}

	if p.Proxy != "" {
		opts = append(opts, grpc.WithContextDialer(newGRPCProxyDialer(p)))
	}

	return grpc.NewClient(p.Address, opts...)
}

func newGRPCProxyDialer(p DialParams) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (conn net.Conn, err error) {
		defer func() {
			if err != nil && conn != nil {
				if err2 := conn.Close(); err2 != nil {
					log.Warning("error when closing connection", err2)
				}
			}
		}()

		proxy, err := url.Parse(p.Proxy)
		if err != nil {
			return nil, errors.Join(errors.New("error parsing the proxy url"), err)
		}

		if proxy.Scheme == "https" {
			cert, err := os.ReadFile(p.ProxyCertPath)
			if err != nil {
				return nil, errors.Join(errors.New("failed to load proxy cert"), err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(cert)

			// No mutual TLS for now
			tlsConfig := tls.Config{RootCAs: caCertPool}
			conn, err = tls.Dial("tcp", proxy.Host, &tlsConfig)
			if err != nil {
				return nil, errors.Join(errors.New("failed to dial the https proxy"), err)
			}
		} else if proxy.Scheme == "http" {
			conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", proxy.Host)
			if err != nil {
				return nil, errors.Join(errors.New("failed to dial the http proxy"), err)
			}
		} else {
			return nil, fmt.Errorf("proxy scheme not supported: %s", proxy.Scheme)
		}

		return httpConnectHandshake(ctx, conn, addr, proxy)
	}
}

func httpConnectHandshake(ctx context.Context, conn net.Conn, server string, proxy *url.URL) (net.Conn, error) {
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: server},
		Header: map[string][]string{"User-Agent": {grpcUA}},
	}
	if t := proxy.User; t != nil {
		u := t.Username()
		p, _ := t.Password()
		req.Header.Add(proxyAuthHeader, "Basic "+basicAuth(u, p))
	}

	req = req.WithContext(ctx)
	if err := req.Write(conn); err != nil {
		return nil, fmt.Errorf("failed to write the HTTP request: %v", err)
	}

	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, req)
	if err != nil {
		return nil, fmt.Errorf("reading server HTTP response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, fmt.Errorf("failed to do connect handshake, status code: %s", resp.Status)
		}
		return nil, fmt.Errorf("failed to do connect handshake, response: %q", dump)
	}

	return &bufConn{Conn: conn, r: r}, nil
}

type bufConn struct {
	net.Conn
	r io.Reader
}

func (c *bufConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

const proxyAuthHeader = "Proxy-Authorization"
const grpcUA = "grpc-go/" + grpc.Version

func printRPCMsg(m Method) {
	if log.Level() > log.DEBUG {
		return
	}

	var str []string
	str = append(str, m.String())

	msgs := m.Message()
	for _, msg := range msgs {
		str = append(str, utils.SPrintBson(msg))
	}
	log.Debugf("%s", str)
}

func bytesToInt32(b []byte) (int32, error) {
	if len(b) != 4 {
		return -1, fmt.Errorf("invalid length: %d", len(b))
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

func ParseInt32(args map[string][]byte, key string, fb int32) int32 {
	ret := fb
	if c, ok := args[key]; ok {
		v, err := bytesToInt32(c)
		if err == nil && v >= 0 {
			ret = v
			log.Debugf("parsed %s=%d", key, v)
		} else {
			log.Warningf("parse error: %s=%d err=%v fallback=%d", key, v, err, fb)
		}
	}
	return ret
}
