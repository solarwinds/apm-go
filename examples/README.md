# Examples

In this directory you will find examples of how to instrument your code using
this library, as well as _de facto_ standard OpenTelemetry instrumentation 
libraries.

Current examples:

 - [http service with a database](http)
 - [grpc client and server](grpc)

For each of these examples, you must provide a `SW_APM_SERVICE_KEY` (as 
documented in the top-level [README](../README.md)), or the library will be 
unable to report traces to SolarWinds Observability.