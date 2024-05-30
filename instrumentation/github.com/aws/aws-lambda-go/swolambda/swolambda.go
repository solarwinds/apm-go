// © 2024 SolarWinds Worldwide, LLC. All rights reserved.
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
