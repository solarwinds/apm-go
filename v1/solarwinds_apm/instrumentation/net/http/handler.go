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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/constants"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"strings"
)

func NewHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ctx := trace.SpanContextFromContext(r.Context()); ctx.IsValid() {
			flags := "00"
			if ctx.IsSampled() {
				flags = "01"
			}
			x := fmt.Sprintf("00-%s-%s-%s", ctx.TraceID().String(), ctx.SpanID().String(), flags)
			exposeHeaders := []string{constants.XTraceHdr}
			w.Header().Add(constants.XTraceHdr, x)

			span := trace.SpanFromContext(r.Context())
			roSpan, ok := span.(sdktrace.ReadOnlySpan)
			if ok {
				attrs := roSpan.Attributes()
				for _, kv := range attrs {
					if kv.Key == constants.SWXTraceOptsResp {
						exposeHeaders = append(exposeHeaders, constants.XTraceOptsRespHdr)
						w.Header().Add(constants.XTraceOptsRespHdr, kv.Value.AsString())
						break
					}
				}
			}

			w.Header().Add("Access-Control-Expose-Headers", strings.Join(exposeHeaders, ","))
		}

		h.ServeHTTP(w, r)
	})
}
