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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TracerSetup() (trace.Tracer, func()) {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(newDummyExporter()),
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
