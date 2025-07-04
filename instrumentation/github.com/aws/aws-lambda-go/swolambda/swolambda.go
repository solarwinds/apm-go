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
	"os"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/swo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	flusher         swo.Flusher
	tracer          trace.Tracer
	initHandlerOnce sync.Once
	warmStart       atomic.Bool
)

type wrappedHandler struct {
	base    lambda.Handler
	fnName  string
	txnName string
	region  string
}

var _ lambda.Handler = &wrappedHandler{}

func (w *wrappedHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	// Note: We need to figure out how to determine `faas.trigger` attribute
	// which is required by semconv
	attrs := []attribute.KeyValue{
		attribute.String("sw.transaction", w.txnName),
		semconv.FaaSColdstart(!warmStart.Swap(true)),
		semconv.FaaSInvokedName(w.fnName),
		semconv.FaaSInvokedProviderAWS,
		semconv.FaaSInvokedRegion(w.region),
	}
	if lc, ok := lambdacontext.FromContext(ctx); !ok {
		log.Error("could not obtain lambda context")
	} else if lc != nil {
		attrs = append(attrs, semconv.FaaSInvocationID(lc.AwsRequestID))
	}
	ctx, span := tracer.Start(ctx, w.fnName, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attrs...))
	defer func() {
		if r := recover(); r != nil {
			defer panic(r)

			switch x := r.(type) {
			case error:
				log.Error("Recovered error:", x.Error())
				span.SetStatus(codes.Error, x.Error())
				span.RecordError(x, trace.WithStackTrace(true))
			case string:
				log.Error("Recovered string:", x)
				span.SetStatus(codes.Error, x)
			default:
				log.Error("Recovered unknown type (%T): %v\n", x, x)
				span.SetStatus(codes.Error, "Panic from unknown type")
			}
		}

		span.End()
		if flusher != nil {
			if err := flusher.Flush(context.Background()); err != nil {
				log.Error("could not flush lambda metrics", err)
			}
		}
	}()
	res, err := w.base.Invoke(ctx, payload)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err, trace.WithStackTrace(true))
	}
	return res, err
}

func WrapHandler(f interface{}) lambda.Handler {
	initHandlerOnce.Do(func() {
		var err error
		if flusher, err = swo.StartLambda(lambdacontext.LogStreamName); err != nil {
			log.Error("could not initialize SWO lambda instrumentation", err)
		}
		tracer = otel.GetTracerProvider().Tracer("swolambda")
	})
	fnName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	txnName := config.GetTransactionName()
	if txnName == "" {
		txnName = fnName
	}
	return &wrappedHandler{
		base:    lambda.NewHandler(f),
		fnName:  fnName,
		txnName: txnName,
		region:  os.Getenv("AWS_REGION"),
	}
}
