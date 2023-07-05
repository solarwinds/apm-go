package utils

import (
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"strings"
	"testing"
)

func TestGetTransactionName(t *testing.T) {
	// Defaults to `unknown`
	var attrs []attribute.KeyValue
	name := ""
	require.Equal(t, "unknown", GetTransactionName(name, attrs))

	// Favors span name
	name = "foo"
	require.Equal(t, name, GetTransactionName(name, attrs))

	// Favors span name over `http.url`
	attrs = append(attrs, attribute.String("http.url", "https://user:pass@example.com/foo/bar"))
	require.Equal(t, name, GetTransactionName(name, attrs))

	// Will use `http.url` when name is blank, and it strips user:pass
	name = ""
	require.Equal(t, "https://example.com/foo/bar", GetTransactionName(name, attrs))

	// Will use `http.route`
	attrs = []attribute.KeyValue{
		attribute.String("http.route", "/foo/bar"),
	}
	require.Equal(t, "/foo/bar", GetTransactionName(name, attrs))

	// Favors `http.route` over `http.url
	attrs = append(attrs, attribute.String("http.url", "https://user:pass@example.com/foo/bar"))
	require.Equal(t, "/foo/bar", GetTransactionName(name, attrs))

	// Does not use an invalid URL
	attrs = []attribute.KeyValue{
		attribute.String("http.url", ":/"),
	}
	require.Equal(t, "unknown", GetTransactionName(name, attrs))

	// Trims spaces
	name = " my transaction "
	attrs = []attribute.KeyValue{
		attribute.String("http.url", "https://user:pass@example.com/foo/bar"),
	}
	require.Equal(t, "my transaction", GetTransactionName(name, attrs))

	// Truncates long transaction names
	name = strings.Repeat("a", 1024)
	expected := strings.Repeat("a", 255)
	require.Equal(t, expected, GetTransactionName(name, attrs))
}
