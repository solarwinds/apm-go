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
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log"
	"net/url"
	"strings"
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
func deriveTransactionName(name string, attrs []attribute.KeyValue) (txnName string) {
	// TODO: add test
	if txnName = config.GetTransactionName(); txnName != "" {
		return
	}

	var httpRoute, httpUrl = "", ""
	for _, attr := range attrs {
		if attr.Key == semconv.HTTPRouteKey {
			httpRoute = attr.Value.AsString()
		} else if attr.Key == semconv.HTTPURLKey {
			httpUrl = attr.Value.AsString()
		}
	}

	if httpRoute != "" {
		txnName = httpRoute
	} else if name != "" {
		txnName = name
	}
	if httpUrl != "" && strings.TrimSpace(txnName) == "" {
		parsed, err := url.Parse(httpUrl)
		if err != nil {
			// We can't import internal logger in the util package, so we default to "log". However, this should be
			// infrequent.
			log.Println("could not parse URL from span", httpUrl)
		} else {
			// Clear user/password
			parsed.User = nil
			txnName = parsed.String()
		}
	}
	txnName = strings.TrimSpace(txnName)
	if txnName == "" {
		txnName = "unknown"
	}

	if len(txnName) > 255 {
		txnName = txnName[:255]
	}
	return
}
