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
	"github.com/XSAM/otelsql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/solarwinds/apm-go/instrumentation/net/http/swohttp"
	"github.com/solarwinds/apm-go/swo"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net/http"
)

func main() {
	// Initialize the SolarWinds APM library
	cb, err := swo.Start(
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

	// Here we use the github.com/XSAM/otelsql instrumentation library that
	// wraps a standard `*sql.DB` handle.
	db, err := otelsql.Open(
		"sqlite3",
		":memory:",
		// The SQL commenter helps associate queries with traces.
		otelsql.WithSQLCommenter(true),
		// We set the Otel semantic convention attribute `db.system` to `sqlite`.
		// otelsql provides standard attributes for many database systems such
		// as MySQL, PostgreSQL, and many others.
		otelsql.WithAttributes(semconv.DBSystemSqlite),
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Create a new handler to respond to any request with the text it was given
	echoHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if text, err := io.ReadAll(req.Body); err != nil {
			// The `trace` package is from the OpenTelemetry Go SDK. Here we
			// retrieve the current span for this request...
			span := trace.SpanFromContext(req.Context())
			// ...so that we can record the error.
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read body")
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			// It's important to inject the request context into the query call
			// because it carries the current trace information.
			row := db.QueryRowContext(req.Context(), "SELECT 1")
			var i int
			if err := row.Scan(&i); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				// If no error, we simply echo back.
				_, _ = w.Write(text)
			}
		}
	})

	mux := http.NewServeMux()
	// Wrap the route handler with otelhttp instrumentation, adding the route tag
	mux.Handle("/echo", otelhttp.WithRouteTag("/echo", echoHandler))
	// Wrap the mux (base handler) with our instrumentation
	http.ListenAndServe(":8080", swohttp.WrapBaseHandler(mux, "server"))
}
