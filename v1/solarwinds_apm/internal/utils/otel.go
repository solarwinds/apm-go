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

package utils

import (
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"log"
	"net/url"
	"strings"
)

// GetTransactionName returns transaction name from given span name and attributes, falling back to "unknown"
func GetTransactionName(name string, attrs []attribute.KeyValue) string {
	var httpRoute, httpUrl, txnName = "", "", ""
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
	return txnName
}

func IsEntrySpan(span sdktrace.ReadOnlySpan) bool {
	parent := span.Parent()
	return !parent.IsValid() || parent.IsRemote()
}
