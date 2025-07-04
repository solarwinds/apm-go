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

package semconv

import (
	otelconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	ExceptionEventName     = otelconv.ExceptionEventName
	ExceptionMessageKey    = otelconv.ExceptionMessageKey
	ExceptionTypeKey       = otelconv.ExceptionTypeKey
	ExceptionStacktraceKey = otelconv.ExceptionStacktraceKey

	HTTPMethodKey     = otelconv.HTTPRequestMethodKey
	HTTPRouteKey      = otelconv.HTTPRouteKey
	HTTPStatusCodeKey = otelconv.HTTPResponseStatusCodeKey
	HTTPURLKey        = otelconv.URLFullKey

	K8SNamespaceNameKey = otelconv.K8SNamespaceNameKey
	K8SPodNameKey       = otelconv.K8SPodNameKey
	K8SPodUIDKey        = otelconv.K8SPodUIDKey

	OTelStatusDescriptionKey = otelconv.OTelStatusDescriptionKey

	ServiceNameKey       = otelconv.ServiceNameKey
	ServiceInstanceIDKey = otelconv.ServiceInstanceIDKey
)

// KeyValues

var (
	OTelStatusCodeOk    = otelconv.OTelStatusCodeOk
	OTelStatusCodeError = otelconv.OTelStatusCodeError

	HTTPRequestMethodGet = otelconv.HTTPRequestMethodGet
)

// Functions

var (
	HTTPStatusCode = otelconv.HTTPResponseStatusCode
	HTTPRoute      = otelconv.HTTPRoute

	K8SNamespaceName = otelconv.K8SNamespaceName
	K8SPodName       = otelconv.K8SPodName
	K8SPodUID        = otelconv.K8SPodUID

	ServiceName = otelconv.ServiceName

	HostID = otelconv.HostID
)
