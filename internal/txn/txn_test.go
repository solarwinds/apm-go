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
	"os"
	"strings"
	"testing"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/testutils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestGetTransactionName(t *testing.T) {
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

func TestDeriveTransactionName(t *testing.T) {
	// Defaults to `unknown`
	var attrs []attribute.KeyValue
	name := ""
	require.Equal(t, "unknown", deriveTransactionName(name, attrs))

	// Favors span name
	name = "foo"
	require.Equal(t, name, deriveTransactionName(name, attrs))

	// Favors span name over `http.url`
	attrs = append(attrs, attribute.String("http.url", "https://user:pass@example.com/foo/bar"))
	require.Equal(t, name, deriveTransactionName(name, attrs))

	// Will use `http.url` when name is blank, and it strips user:pass
	name = ""
	require.Equal(t, "https://example.com/foo/bar", deriveTransactionName(name, attrs))

	// Will use `http.route`
	attrs = []attribute.KeyValue{
		attribute.String("http.route", "/foo/bar"),
	}
	require.Equal(t, "/foo/bar", deriveTransactionName(name, attrs))

	// Favors `http.route` over `http.url
	attrs = append(attrs, attribute.String("http.url", "https://user:pass@example.com/foo/bar"))
	require.Equal(t, "/foo/bar", deriveTransactionName(name, attrs))

	// Does not use an invalid URL
	attrs = []attribute.KeyValue{
		attribute.String("http.url", ":/"),
	}
	require.Equal(t, "unknown", deriveTransactionName(name, attrs))

	// Trims spaces
	name = " my transaction "
	attrs = []attribute.KeyValue{
		attribute.String("http.url", "https://user:pass@example.com/foo/bar"),
	}
	require.Equal(t, "my transaction", deriveTransactionName(name, attrs))

	// Truncates long transaction names
	name = strings.Repeat("a", 1024)
	expected := strings.Repeat("a", 255)
	require.Equal(t, expected, deriveTransactionName(name, attrs))
}

func TestDeriveTxnFromEnv(t *testing.T) {
	envTxn := "env-provided"
	name := "span name"
	var attrs []attribute.KeyValue
	// `SW_APM_TRANSACTION_NAME` only takes effect in Lambda
	require.NoError(t, os.Setenv("SW_APM_TRANSACTION_NAME", envTxn))
	config.Load()
	require.Equal(t, "", config.GetTransactionName())
	require.Equal(t, "span name", deriveTransactionName(name, attrs))

	require.NoError(t, os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "foo"))
	require.NoError(t, os.Setenv("LAMBDA_TASK_ROOT", "bar"))
	defer func() {
		_ = os.Unsetenv("SW_APM_TRANSACTION_NAME")
		_ = os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
		_ = os.Unsetenv("LAMBDA_TASK_ROOT")
	}()
	config.Load()
	require.Equal(t, envTxn, config.GetTransactionName())
	require.Equal(t, envTxn, deriveTransactionName(name, attrs))
}
func TestDeriveTxnFromEnvTruncated(t *testing.T) {
	envTxn := strings.Repeat("a", 1024)
	expected := strings.Repeat("a", 255)
	name := "span name"
	var attrs []attribute.KeyValue
	require.NoError(t, os.Setenv("SW_APM_TRANSACTION_NAME", envTxn))
	require.NoError(t, os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "foo"))
	require.NoError(t, os.Setenv("LAMBDA_TASK_ROOT", "bar"))
	defer func() {
		_ = os.Unsetenv("SW_APM_TRANSACTION_NAME")
		_ = os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
		_ = os.Unsetenv("LAMBDA_TASK_ROOT")
	}()
	config.Load()
	require.Equal(t, envTxn, config.GetTransactionName())
	require.Equal(t, expected, deriveTransactionName(name, attrs))
}
