// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
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

package swo

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestStartLambdaAndFlush(t *testing.T) {
	// Start a local TCP listener to accept gRPC connections
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { require.NoError(t, lis.Close()) }()

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://"+lis.Addr().String())
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("SW_APM_SERVICE_KEY", "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:test")

	// StartLambda return Flusher object
	flusher, err := StartLambda("test-log-stream")
	require.NoError(t, err)
	require.NotNil(t, flusher)
	// require.NoError(t, flusher.Flush(context.Background()))
	flushCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = flusher.Flush(flushCtx) //nolint:errcheck — export fails without server, expected

	// Verify resource attributes set by StartLambda.
	// Register a SpanRecorder, then emit a span with a local sampled parent so
	// the oboe sampler passes it through (non-remote sampled parent → always sample).
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok, "global tracer provider should be *sdktrace.TracerProvider")

	rec := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(rec)

	// Use a local (non-remote), sampled parent span context so the oboe sampler
	// defers to AlwaysSample, ensuring the span is recorded.
	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
		Remote:     false,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), parent)
	_, span := otel.Tracer("test").Start(ctx, "probe")
	span.End()

	ended := rec.Ended()
	require.Len(t, ended, 1)

	attrs := make(map[string]string)
	for _, kv := range ended[0].Resource().Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}
	require.Equal(t, "apm", attrs["sw.data.module"])
	require.Equal(t, "test-log-stream", attrs["faas.instance"])
}

// TestLambdaServiceNameFallback covers the AWS Lambda service.name precedence special case:
// OTEL_SERVICE_NAME > OTEL_RESOURCE_ATTRIBUTES > AWS_LAMBDA_FUNCTION_NAME > SDK default.
func TestLambdaServiceNameFallback(t *testing.T) {
	t.Run("falls back to AWS_LAMBDA_FUNCTION_NAME when base has SDK default", func(t *testing.T) {
		t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "name-from-lambda-env-var")
		base := resource.NewSchemaless(attribute.String("service.name", "unknown_service:go"))
		r := lambdaServiceNameFallback(base)
		val, ok := r.Set().Value(attribute.Key("service.name"))
		require.True(t, ok)
		require.Equal(t, "name-from-lambda-env-var", val.AsString())
	})

	t.Run("does not override an explicit OTEL_SERVICE_NAME/OTEL_RESOURCE_ATTRIBUTES value", func(t *testing.T) {
		t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "name-from-lambda-env-var")
		base := resource.NewSchemaless(attribute.String("service.name", "name-from-env-var"))
		r := lambdaServiceNameFallback(base)
		_, ok := r.Set().Value(attribute.Key("service.name"))
		require.False(t, ok, "must not set service.name when base already has an explicit value")
	})

	t.Run("leaves SDK default in place when AWS_LAMBDA_FUNCTION_NAME unset", func(t *testing.T) {
		base := resource.NewSchemaless(attribute.String("service.name", "unknown_service:go"))
		r := lambdaServiceNameFallback(base)
		_, ok := r.Set().Value(attribute.Key("service.name"))
		require.False(t, ok)
	})
}
