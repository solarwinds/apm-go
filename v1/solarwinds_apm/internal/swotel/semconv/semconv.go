package semconv

import (
	otelconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

const (
	ExceptionEventName     = otelconv.ExceptionEventName
	ExceptionMessageKey    = otelconv.ExceptionMessageKey
	ExceptionTypeKey       = otelconv.ExceptionTypeKey
	ExceptionStacktraceKey = otelconv.ExceptionStacktraceKey

	HTTPMethodKey     = otelconv.HTTPMethodKey
	HTTPRouteKey      = otelconv.HTTPRouteKey
	HTTPStatusCodeKey = otelconv.HTTPStatusCodeKey
	HTTPURLKey        = otelconv.HTTPURLKey

	K8SNamespaceNameKey = otelconv.K8SNamespaceNameKey
	K8SPodNameKey       = otelconv.K8SPodNameKey
	K8SPodUIDKey        = otelconv.K8SPodUIDKey

	ServiceNameKey = otelconv.ServiceNameKey
)

// Functions

var (
	HTTPStatusCode = otelconv.HTTPStatusCode
	HTTPMethod     = otelconv.HTTPMethod
	HTTPRoute      = otelconv.HTTPRoute

	K8SNamespaceName = otelconv.K8SNamespaceName
	K8SPodName       = otelconv.K8SPodName
	K8SPodUID        = otelconv.K8SPodUID

	ServiceName = otelconv.ServiceName
)
