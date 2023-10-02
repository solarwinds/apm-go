# Examples

In this directory you will find examples of how to instrument your code using
this library, as well at _de facto_ standard OpenTelemetry instrumentation 
libraries.

Current examples:

 - [http](http)
 - [db](http) &ndash; The database example is located within the [http](http)
   example as it shows how to pass the trace data down the stack via 
   [`context.Context`](https://pkg.go.dev/context#Context)
 - [grpc](grpc)

For each of these examples, you must provide a `SW_APM_SERVICE_KEY` (as 
documented in the top-level [README](../README.md)), or the library will be 
unable to report traces to SolarWinds Observability.