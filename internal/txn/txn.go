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
	"os"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// GetTransactionName retrieves the custom transaction name if it exists, otherwise calls deriveTransactionName
func GetTransactionName(span sdktrace.ReadOnlySpan) string {
	if txnName := entryspans.GetTransactionName(span.SpanContext().TraceID()); txnName != "" {
		return txnName
	} else {
		return deriveTransactionName(span.Name(), span.Attributes())
	}
}

// deriveTransactionName returns transaction name from given span name and attributes, falling back to "unknown"
func deriveTransactionName(spanName string, attrs []attribute.KeyValue) string {
	// First priority: Check configuration
	txnName := config.GetTransactionName()

	// Second priority: Environment variables
	if txnName == "" {
		if valFromEnv := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); valFromEnv != "" {
			txnName = valFromEnv
		}
	}

	// Third priority: Derive from span attributes
	if txnName == "" {
		var faasName, httpRoute, urlPath string
		for _, attr := range attrs {
			switch attr.Key {
			case semconv.FaaSNameKey:
				faasName = attr.Value.AsString()
			case semconv.HTTPRouteKey:
				httpRoute = attr.Value.AsString()
			case semconv.URLPathKey:
				urlPath = attr.Value.AsString()
			case semconv.HTTPTargetKey:
				urlPath = attr.Value.AsString()
			}
		}

		// Priority: faas.name > http.route > url.path/http.target
		if faasName != "" {
			txnName = faasName
		} else if httpRoute != "" {
			txnName = httpRoute
		} else if urlPath != "" {
			txnName = extractTransactionNameFromPath(urlPath)
		}
	}

	// Fourth priority: Use span name if nothing else is available
	if txnName == "" {
		txnName = spanName
	}

	// Final fallback: use "unknown"
	txnName = strings.TrimSpace(txnName)
	if txnName == "" {
		txnName = "unknown"
	}

	if len(txnName) > 255 {
		txnName = txnName[:255]
	}
	return txnName
}

// extractTransactionNameFromPath extracts up to 2 path segments from a URL path,
// ignoring query parameters and fragments
func extractTransactionNameFromPath(urlPath string) string {
	// Drop query parameters and fragment
	if idx := strings.IndexAny(urlPath, "?#"); idx != -1 {
		urlPath = urlPath[:idx]
	}

	// Split path into segments (limit to 4 to get first 2 non-empty segments)
	segments := strings.SplitN(urlPath, "/", 4)

	switch len(segments) {
	case 0, 1:
		return urlPath
	case 2:
		// Path has 1 segment: /segment
		return "/" + segments[1]
	default:
		// Path has 2+ segments: take first 2
		return "/" + segments[1] + "/" + segments[2]
	}
}
