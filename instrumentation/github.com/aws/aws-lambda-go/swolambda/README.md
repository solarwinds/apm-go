# Instrumentation for AWS Lambda

This package instruments the AWS Lambda `Handler` interface. 

## Usage

### Add the `Otelcol` extension layer

Follow the [SolarWinds Observability 
documentation](https://documentation.solarwinds.com/en/success_center/observability/content/intro/services/aws-lambda-overview.htm) 
to add the Otelcol extension layer. 

**Note**: Unlike other languages, Golang does not require an additional 
extension, so the "Instrumentation extension" section on that page does not 
apply.

### Modify your code

First, install the dependency:
```shell
go get -u github.com/solarwinds/apm-go/instrumentation/github.com/aws/aws-lambda-go/swolambda
```

Then, wrap your handler with `swolambda.WrapHandler`:

```go
package main
import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/solarwinds/apm-go/instrumentation/github.com/aws/aws-lambda-go/swolambda"
)

// Example incoming type
type MyEvent struct {}

// This is an example handler, yours may have a different signature and a 
// different name. It will work ass long as it adheres to what the Lambda SDK
// expects. (See "Valid handler signatures"[0])
// [0] https://docs.aws.amazon.com/lambda/latest/dg/golang-handler.html
func ExampleHandler(ctx context.Context, event *MyEvent) (string, error) {
	return "hello world", nil
}
func main() {
	// We wrap our handler here and pass the result to `lambda.Start`
	lambda.Start(swolambda.WrapHandler(ExampleHandler))
}
```

Now that you've instrumented your code, you should be able to send requests and
see the resulting metrics and traces in SWO.