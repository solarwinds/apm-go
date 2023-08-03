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

package swohttp

import (
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/swotel"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"strings"
)

const (
	XTraceHdr         = "X-Trace"
	XTraceOptsRespHdr = "X-Trace-Options-Response"
)

func Wrap(h http.Handler, operation string) http.Handler {
	// Wrap with our instrumentation
	h = NewHandler(h)
	// Wrap with Otel
	return otelhttp.NewHandler(h, operation)
}

func NewHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ctx := trace.SpanContextFromContext(r.Context()); ctx.IsValid() {
			flags := "00"
			if ctx.IsSampled() {
				flags = "01"
			}
			x := fmt.Sprintf("00-%s-%s-%s", ctx.TraceID().String(), ctx.SpanID().String(), flags)
			exposeHeaders := []string{XTraceHdr}
			w.Header().Add(XTraceHdr, x)

			ts := ctx.TraceState()
			resp, err := swotel.GetInternalState(ts, swotel.XTraceOptResp)
			if err != nil {
				log.Debugf("Could not get xtrace opt resp header: %s", err)
			}
			if resp != "" {
				exposeHeaders = append(exposeHeaders, XTraceOptsRespHdr)
				w.Header().Add(XTraceOptsRespHdr, resp)
			}

			w.Header().Add("Access-Control-Expose-Headers", strings.Join(exposeHeaders, ","))
		}

		h.ServeHTTP(w, r)
	})
}
