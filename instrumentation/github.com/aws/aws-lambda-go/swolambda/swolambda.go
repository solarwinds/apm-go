package swolambda

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/swo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"os"
	"sync"
)

var flusher swo.Flusher
var tracer trace.Tracer
var once sync.Once

type wrappedHandler struct {
	base lambda.Handler
}

var _ lambda.Handler = &wrappedHandler{}

func (w *wrappedHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	var attrs []attribute.KeyValue
	if lc, ok := lambdacontext.FromContext(ctx); !ok {
		log.Error("could not obtain lambda context")
	} else if lc != nil {
		attrs = append(attrs, semconv.FaaSInvocationID(lc.AwsRequestID))
	}
	name := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	ctx, span := tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attrs...))
	defer func() {
		span.End()
		if flusher != nil {
			if err := flusher.Flush(context.Background()); err != nil {
				log.Error("could not flush lambda metrics", err)
			}
		}
	}()
	return w.base.Invoke(ctx, payload)
}

func WrapHandler(f interface{}) lambda.Handler {
	once.Do(func() {
		var err error
		if flusher, err = swo.StartLambda(lambdacontext.LogStreamName); err != nil {
			log.Error("could not initialize SWO lambda instrumentation", err)
		}
		tracer = otel.GetTracerProvider().Tracer("swolambda")
	})
	return &wrappedHandler{
		base: lambda.NewHandler(f),
	}
}
