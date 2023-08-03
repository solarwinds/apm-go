package main

import (
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm"
	swohttp "github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/instrumentation/net/http"
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
