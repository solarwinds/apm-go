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

package main

import (
	swohttp "github.com/solarwindscloud/solarwinds-apm-go/instrumentation/net/http"
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net/http"
)

func main() {
	// Initialize the solarwinds_apm library
	cb, err := solarwinds_apm.Start(
		// Optionally add service-level resource attributes
		semconv.ServiceName("my-service"),
		semconv.ServiceVersion("v0.0.1"),
		attribute.String("environment", "testing"),
	)
	if err != nil {
		// Handle error
	}
	// This function returned from `Start()` will tell the apm library to
	// shut down, often deferred until the end of `main()`.
	defer cb()

	// Create a new handler to respond to any request with the text it was given
	echoHandler := swohttp.Wrap(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if text, err := io.ReadAll(req.Body); err != nil {
			// The `trace` package is from the OpenTelemetry Go SDK. Here we
			// retrieve the current span for this request...
			span := trace.SpanFromContext(req.Context())
			// ...so that we can record the error.
			span.RecordError(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			// If no error, we simply echo back.
			_, _ = w.Write(text)
		}
	}), "echo")
	server := &http.Server{Addr: ":8080"}
	http.Handle("/echo", echoHandler)

	server.ListenAndServe()
}
