// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package solarwinds_apm

import (
	"net/http"

	"context"
)

// HTTPClientSpan is a Span that aids in reporting HTTP client requests.
//
//	req, err := http.NewRequest("GET", "http://example.com", nil)
//	l := solarwinds_apm.BeginHTTPClientSpan(ctx, httpReq)
//	defer l.End()
//	// ...
//	resp, err := client.Do(req)
//	l.AddHTTPResponse(resp, err)
//	// ...
type HTTPClientSpan struct{ Span }

// BeginHTTPClientSpan stores trace metadata in the headers of an HTTP client request, allowing the
// trace to be continued on the other end. It returns a Span that must have End() called to
// benchmark the client request, and should have AddHTTPResponse(r, err) called to process response
// metadata.
func BeginHTTPClientSpan(ctx context.Context, req *http.Request) HTTPClientSpan {
	if req != nil {
		l := BeginRemoteURLSpan(ctx, "http.Client", req.URL.String(), "HTTPMethod", req.Method)
		req.Header.Set(HTTPHeaderName, l.MetadataString())
		return HTTPClientSpan{Span: l}
	}
	return HTTPClientSpan{Span: nullSpan{}}
}

// AddHTTPResponse adds information from http.Response to this span. It will also check the HTTP
// response headers and propagate any valid distributed trace context from the end of the HTTP
// server's span to this one.
func (l HTTPClientSpan) AddHTTPResponse(resp *http.Response, err error) {
	if l.ok() {
		if err != nil {
			l.Err(err)
		}
		if resp != nil {
			l.AddEndArgs(keyRemoteStatus, resp.StatusCode, keyContentLength, resp.ContentLength)
			if md := resp.Header.Get(HTTPHeaderName); md != "" {
				l.AddEndArgs(keyEdge, md)
			}
		}
	}
}
