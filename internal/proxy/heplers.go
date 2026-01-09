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

package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/solarwinds/apm-go/internal/log"
	"google.golang.org/grpc"
)

const proxyAuthHeader = "Proxy-Authorization"
const grpcUA = "grpc-go/" + grpc.Version

type ProxyOptions struct {
	Proxy         string
	ProxyCertPath string
}

type bufConn struct {
	net.Conn
	r io.Reader
}

// Add this method to properly read from buffered reader
func (b *bufConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func NewGRPCProxyDialer(p ProxyOptions) func(context.Context, string) (net.Conn, error) {
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

		switch proxy.Scheme {
		case "https":
			caCertPool, err := getProxyCertPool(p.ProxyCertPath)
			if err != nil {
				return nil, errors.Join(errors.New("failed to load proxy cert"), err)
			}
			// No mutual TLS for now
			tlsConfig := tls.Config{RootCAs: caCertPool}
			conn, err = tls.Dial("tcp", proxy.Host, &tlsConfig)
			if err != nil {
				return nil, errors.Join(errors.New("failed to dial the https proxy"), err)
			}
		case "http":
			conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", proxy.Host)
			if err != nil {
				return nil, errors.Join(errors.New("failed to dial the http proxy"), err)
			}
		default:
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

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func NewHttpTransport(p ProxyOptions) (*http.Transport, error) {
	parsedProxyURL, err := url.Parse(p.Proxy)
	if err != nil {
		log.Errorf("invalid proxy URL %s: %v", p.Proxy, err)
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedProxyURL),
	}

	if certPath := p.ProxyCertPath; certPath != "" {
		certPool, err := getProxyCertPool(certPath)
		if err != nil {
			return nil, err
		}
		if certPool != nil {
			transport.TLSClientConfig = &tls.Config{
				RootCAs: certPool,
			}
		}
	}

	return transport, nil
}

func getProxyCertPool(certPath string) (*x509.CertPool, error) {
	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to load proxy cert"), err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(cert)
	return caCertPool, nil
}
