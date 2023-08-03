# SolarWinds APM Go

## Notice

⚠ This is a work-in-progress, originally forked from the
[appoptics-apm-go](https://github.com/appoptics/appoptics-apm-go) repository.

## Getting started

ℹ️ Check out the [http example](examples/http/README.md) to see how easy it is
to get started. The following code snippets are pulled directly from it.

To use in your Go project:

### Initialize the library

```go
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
```

### Instrument your code

Many packages have instrumentation-enabled versions. We provide a simple 
wrapper for `net/http` server requests. Here's an example:

```go
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
```

There are many instrumented libraries available. Here are the libraries we
currently support:

  * Any of the instrumentation in [opentelemetry-go-contrib
](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation)
    * _Caveat_: For `net/http` servers, it's best to use our wrapper (as seen
      in the above example) to correctly attribute distributed trace data.
  * For SQL: [XSAM/otelsql](https://github.com/XSAM/otelsql)

OpenTelemetry provides a
[registry](https://opentelemetry.io/ecosystem/registry/?language=go&component=instrumentation)
for these libraries, just note that each is at a different maturity
level, as the OpenTelemetry landscape is developing at a rapid pace.

You may also [manually 
instrument](https://opentelemetry.io/docs/instrumentation/go/manual/) your code
using the OpenTelemetry SDK and it will be properly propagated to SolarWinds
Observability.

### Configuration

The only environment variable you need to set before kicking off is the service key:

| Variable Name       | Required           |  Description |
|---------------------| ------------------ |  ----------- |
| SW_APM_SERVICE_KEY  |Yes|The service key identifies the service being instrumented within your Organization. It should be in the form of ``<api token>:<service name>``.|

## Compatibility

We support the same environments as
[OpenTelemetry-Go](https://github.com/open-telemetry/opentelemetry-go#compatibility).

## License

Copyright (C) 2023 SolarWinds, LLC

Released under [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0)
