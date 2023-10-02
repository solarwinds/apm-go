# Instrumented HTTP and Database Example

[`main.go`](main.go) shows how simple it is to get started.

To run, set the `SW_APM_SERVICE_KEY` environment variable, then run `go run .`
in this directory. For example:

```shell
SW_APM_SERVICE_KEY="${SW_APM_TOKEN}:golang-testbed" go run .
```

If the agent successfully connects, you will see a log line like this:

```
[solarwinds_apm] Got dynamic settings. The SolarWinds Observability APM agent (0x140001007e0) is ready.
```

Call the http service like so:

```shell
% curl -ik -d "hello world" http://localhost:8080/echo
HTTP/1.1 200 OK
Access-Control-Expose-Headers: X-Trace
X-Trace: 00-61b3d30b3695fc559ed760d75460c4ed-09c1e095da6f1c51-01
Date: Thu, 03 Aug 2023 15:39:44 GMT
Content-Length: 11
Content-Type: text/plain; charset=utf-8

hello world%
```

The `X-Trace` response header is in the [W3C Trace Context Traceparent Header
](https://www.w3.org/TR/trace-context/#traceparent-header) format. The `01` at
the end indicates the request was sampled, meaning the trace will show up in
your SolarWinds APM dashboard. `00` at the end will indicate _not_ sampled.