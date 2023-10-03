# Instrumented gRPC example 

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

This is a trivial service that only calls itself. If you provided a valid 
service key, you should see traces in SWO that show the gRPC client calling the 
gRPC server.