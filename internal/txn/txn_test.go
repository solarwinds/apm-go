// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
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

package txn

import (
	"context"
	"strings"
	"testing"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/testutils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestGetTransactionWhenUpdatedWithAPI(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()

	ctx := context.Background()
	_, span := tr.Start(ctx, "derived")
	roSpan, ok := span.(trace.ReadWriteSpan)
	err := entryspans.Push(roSpan)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "derived", GetTransactionName(roSpan))
	err = entryspans.SetTransactionName(span.SpanContext().TraceID(), "custom")
	require.NoError(t, err)
	require.Equal(t, "custom", GetTransactionName(roSpan))
}

func TestTransactionNamePrecedenceOrder(t *testing.T) {
	// Priority order:
	// 1. config.GetTransactionName() (SW_APM_TRANSACTION_NAME in Lambda)
	// 2. AWS_LAMBDA_FUNCTION_NAME environment variable (only if config is empty)
	// 3. Span attributes (faas.name > http.route > url.path)
	// 4. Span name
	// 5. "unknown"

	tests := []struct {
		name     string
		envVars  map[string]string
		spanName string
		attrs    []attribute.KeyValue
		expected string
	}{
		{
			name:     "lambda env var used when config is empty",
			envVars:  map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "lambda-func"},
			spanName: "my-span",
			attrs: []attribute.KeyValue{
				{Key: semconv.FaaSNameKey, Value: attribute.StringValue("attr-lambda")},
				{Key: semconv.HTTPRouteKey, Value: attribute.StringValue("/api/route")},
				{Key: semconv.URLPathKey, Value: attribute.StringValue("/users/123")},
			},
			expected: "lambda-func",
		},
		{
			name:     "faas.name beats http.route and url.path",
			spanName: "my-span",
			attrs: []attribute.KeyValue{
				{Key: semconv.FaaSNameKey, Value: attribute.StringValue("attr-lambda")},
				{Key: semconv.HTTPRouteKey, Value: attribute.StringValue("/api/route")},
				{Key: semconv.URLPathKey, Value: attribute.StringValue("/users/123")},
			},
			expected: "attr-lambda",
		},
		{
			name:     "http.route beats url.path and span name",
			spanName: "my-span",
			attrs: []attribute.KeyValue{
				{Key: semconv.HTTPRouteKey, Value: attribute.StringValue("/api/route")},
				{Key: semconv.URLPathKey, Value: attribute.StringValue("/users/123")},
			},
			expected: "/api/route",
		},
		{
			name:     "url.path beats span name",
			spanName: "my-span",
			attrs:    []attribute.KeyValue{{Key: semconv.URLPathKey, Value: attribute.StringValue("/users/123")}},
			expected: "/users/123",
		},
		{
			name:     "span name beats unknown",
			spanName: "my-span",
			expected: "my-span",
		},
		{
			name:     "defaults to unknown when nothing provided",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result := deriveTransactionName(tt.spanName, tt.attrs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDeriveTransactionName_PathParsing(t *testing.T) {
	pathTests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "single segment",
			path:     "/search",
			expected: "/search",
		},
		{
			name:     "two segments",
			path:     "/users/123",
			expected: "/users/123",
		},
		{
			name:     "three segments - takes first two",
			path:     "/api/users/123",
			expected: "/api/users",
		},
		{
			name:     "multiple segments - takes first two",
			path:     "/api/users/123/details",
			expected: "/api/users",
		},
		{
			name:     "drops query parameters",
			path:     "/users/123?id=5&name=test",
			expected: "/users/123",
		},
		{
			name:     "drops fragment",
			path:     "/search#results",
			expected: "/search",
		},
		{
			name:     "drops query and fragment",
			path:     "/api/v1?page=2#section",
			expected: "/api/v1",
		},
	}

	attributeKeys := []struct {
		name string
		key  attribute.Key
	}{
		{name: "url.path", key: semconv.URLPathKey},
		{name: "http.target", key: semconv.HTTPTargetKey},
	}

	for _, attrTest := range attributeKeys {
		t.Run(attrTest.name, func(t *testing.T) {
			for _, tt := range pathTests {
				t.Run(tt.name, func(t *testing.T) {
					result := deriveTransactionName("", []attribute.KeyValue{{Key: attrTest.key, Value: attribute.StringValue(tt.path)}})
					require.Equal(t, tt.expected, result)
				})
			}
		})
	}
}

func TestDeriveTransactionName_EdgeCases(t *testing.T) {
	t.Run("trims spaces from span name", func(t *testing.T) {
		result := deriveTransactionName("  my transaction  ", nil)
		require.Equal(t, "my transaction", result)
	})

	t.Run("truncates long transaction names", func(t *testing.T) {
		longName := strings.Repeat("a", 1024)
		expected := strings.Repeat("a", 255)
		result := deriveTransactionName(longName, nil)
		require.Equal(t, expected, result)
	})

	t.Run("uses span name when no attributes", func(t *testing.T) {
		result := deriveTransactionName("my-span", nil)
		require.Equal(t, "my-span", result)
	})

	t.Run("defaults to unknown with empty span name and no attributes", func(t *testing.T) {
		result := deriveTransactionName("", nil)
		require.Equal(t, "unknown", result)
	})
}

func TestDeriveTransactionName_ConfigIntegration(t *testing.T) {
	t.Run("config takes priority over lambda env var", func(t *testing.T) {
		envTxn := "env-provided"
		t.Setenv("SW_APM_TRANSACTION_NAME", envTxn)
		t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "foo")
		t.Setenv("LAMBDA_TASK_ROOT", "bar")
		config.Load()

		require.Equal(t, envTxn, config.GetTransactionName())
		// Config transaction name takes priority over AWS_LAMBDA_FUNCTION_NAME
		require.Equal(t, envTxn, deriveTransactionName("span name", nil))
	})

	t.Run("lambda env var used when config is not set", func(t *testing.T) {
		t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-lambda")
		t.Setenv("LAMBDA_TASK_ROOT", "bar")
		config.Load()

		require.Equal(t, "", config.GetTransactionName())
		// AWS_LAMBDA_FUNCTION_NAME is used when config is empty
		require.Equal(t, "my-lambda", deriveTransactionName("span name", nil))
	})
}
