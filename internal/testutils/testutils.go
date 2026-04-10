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

package testutils

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const SpanIdHex = "0123456789abcdef"
const TraceIdHex = "44444444444444443333333333333333"

var SpanID, err1 = trace.SpanIDFromHex(SpanIdHex)
var TraceID, err2 = trace.TraceIDFromHex(TraceIdHex)

func init() {
	for _, err := range []error{err1, err2} {
		if err != nil {
			log.Fatal("Fatal error: ", err)
		}
	}
}

func TracerSetup() (trace.Tracer, func()) {
	return TracerWithExporter(newDummyExporter())
}

func TracerWithExporter(e sdktrace.SpanExporter) (trace.Tracer, func()) {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(e),
		sdktrace.WithSampler(newDummySampler()),
	)
	otel.SetTracerProvider(tp)
	tr := otel.Tracer(
		"foo123",
		trace.WithInstrumentationVersion("123"),
		trace.WithSchemaURL("https://www.schema.url/foo123"),
	)

	return tr, func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
		}
	}

}

type dummySampler struct{}

func (ds *dummySampler) ShouldSample(sdktrace.SamplingParameters) sdktrace.SamplingResult {
	return sdktrace.SamplingResult{
		Decision: sdktrace.RecordAndSample,
	}
}

func (ds *dummySampler) Description() string {
	return "Dummy Sampler"
}

func newDummySampler() sdktrace.Sampler {
	return &dummySampler{}
}

type dummyExporter struct{}

func newDummyExporter() *dummyExporter {
	return &dummyExporter{}
}

func (de *dummyExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (de *dummyExporter) Shutdown(context.Context) error {
	return nil
}

func Srv(t *testing.T, response string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, err := fmt.Fprint(w, response)
		require.NoError(t, err)
	}))
}

// Setenv Returns a callback for use with defer
func Setenv(t *testing.T, k string, v string) func() {
	require.NoError(t, os.Setenv(k, v))
	return func() {
		require.NoError(t, os.Unsetenv(k))
	}
}

// WriteUUIDFile creates a temporary file containing the given UUID string and
// registers t.Cleanup to remove it. Returns the file path.
func WriteUUIDFile(t *testing.T, id uuid.UUID) string {
	t.Helper()
	f, err := os.CreateTemp("", "uams-test-*")
	require.NoError(t, err)
	_, err = f.WriteString(id.String())
	require.NoError(t, err)
	require.NoError(t, f.Close())
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
	return f.Name()
}

// UamsClientResponse returns a JSON response body matching the UAMS HTTP
// endpoint format with the given UUID as the uamsclient_id.
func UamsClientResponse(id uuid.UUID) string {
	return fmt.Sprintf(`{"is_registered":true,"otel_endpoint_access":false,"usc_connectivity":true,"uamsclient_id":"%s"}`, id.String())
}
