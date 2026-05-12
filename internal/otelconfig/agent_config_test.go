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

package otelconfig

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	otelglobal "go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TestStartWithOtelConfEndpointAndHeaders calls StartWithOtelConf with a YAML
// config that declares both tracer_provider and meter_provider (no processors /
// no readers), verifying exporter endpoint, auth headers, SWO sampler, and
// SWO resource attributes.
func TestStartWithOtelConfEndpointAndHeaders(t *testing.T) {
	lis := setupLocalListener(t)
	endpoint := lis.Addr().String()

	startTestAgent(t, fmt.Sprintf(`file_format: "1.0"
tracer_provider: {}
meter_provider: {}
propagator:
  composite:
    - tracecontext: {}
    - baggage: {}
instrumentation/development:
  go:
    solarwinds:
      ServiceKey: "%s"
      Collector: "%s"
      RuntimeMetrics: false
`, testServiceKey, endpoint))

	// --- tracer provider ---
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok, "expected *sdktrace.TracerProvider, got %T", otel.GetTracerProvider())
	traceDetails := tracerProviderExporterDeepDetails(tp)
	var batchDetail *traceExporterDetail
	for i, d := range traceDetails {
		t.Logf("  processor=%s exporter=%s client=%s endpoint=%q headers=%q",
			d.ProcessorType, d.ExporterType, d.ClientType, d.Endpoint, d.Headers)
		if strings.Contains(d.ProcessorType, "batchSpanProcessor") {
			batchDetail = &traceDetails[i]
		}
	}
	require.NotNil(t, batchDetail, "expected a batchSpanProcessor in the tracer provider")
	assert.Equal(t, endpoint, batchDetail.Endpoint)
	assert.Contains(t, batchDetail.Headers, testHeaderToken)

	// sampler must be (or wrap) the SWO sampler
	samplerDesc := tracerProviderSamplerDescription(tp)
	t.Logf("sampler description: %s", samplerDesc)
	assert.Contains(t, samplerDesc, "SolarWinds APM Sampler")

	// resource must carry SWO-required attributes
	resAttrs := resourceAttrsFromProvider(tp)
	t.Logf("resource attrs: %v", resAttrs)
	assert.Equal(t, "apm", resAttrs["sw.data.module"])
	assert.NotEmpty(t, resAttrs["sw.apm.version"])

	// --- meter provider ---
	meterDetails := meterProviderReaderDeepDetails(otel.GetMeterProvider())
	var periodicDetail *meterReaderDetail
	for i, d := range meterDetails {
		t.Logf("  reader=%s exporter=%s endpoint=%q", d.ReaderType, d.ExporterType, d.Endpoint)
		if strings.Contains(d.ReaderType, "PeriodicReader") {
			periodicDetail = &meterDetails[i]
		}
	}
	require.NotNil(t, periodicDetail, "expected a PeriodicReader in the meter provider")
	assert.Equal(t, endpoint, periodicDetail.Endpoint)
	assert.Contains(t, periodicDetail.Headers, testHeaderToken)

	// --- logger provider (noop — no logger_provider section in YAML) ---
	_, isNoopLP := otelglobal.GetLoggerProvider().(lognoop.LoggerProvider)
	assert.True(t, isNoopLP, "expected noop LoggerProvider, got %T", otelglobal.GetLoggerProvider())

	// --- propagator ---
	fields := otel.GetTextMapPropagator().Fields()
	assert.Contains(t, fields, "traceparent")
	assert.Contains(t, fields, "tracestate")
}

// TestStartWithOtelConfWithUserProvidedProcessorReader verifies that when the
// user declares their own processors and readers in the YAML the SWO agent does
// not override the exporters. The SWO sampler, propagator, and resource
// attributes must still be applied.
func TestStartWithOtelConfWithUserProvidedProcessorReader(t *testing.T) {
	lis := setupLocalListener(t)
	endpoint := lis.Addr().String()

	startTestAgent(t, fmt.Sprintf(`file_format: "1.0"
tracer_provider:
  processors:
    - batch:
        exporter:
          otlp_grpc:
            endpoint: "http://196.162.9.776:4317"
meter_provider:
  readers:
    - periodic:
        interval: 10000
        exporter:
          otlp_grpc:
            endpoint: "http://196.162.9.776:4317"
            temporality_preference: cumulative
propagator:
  composite:
    - tracecontext: {}
    - baggage: {}
instrumentation/development:
  go:
    solarwinds:
      ServiceKey: "%s"
      Collector: "%s"
      RuntimeMetrics: false
`, testServiceKey, endpoint))

	// --- tracer provider: user's batch processor must use the user's endpoint ---
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok, "expected *sdktrace.TracerProvider, got %T", otel.GetTracerProvider())
	traceDetails := tracerProviderExporterDeepDetails(tp)
	var batchDetail *traceExporterDetail
	for i, d := range traceDetails {
		t.Logf("  processor=%s exporter=%s endpoint=%q headers=%q",
			d.ProcessorType, d.ExporterType, d.Endpoint, d.Headers)
		if strings.Contains(d.ProcessorType, "batchSpanProcessor") {
			batchDetail = &traceDetails[i]
		}
	}
	assert.Equal(t, batchDetail.Endpoint, "196.162.9.776:4317")
	assert.NotContains(t, batchDetail.Headers, testHeaderToken)

	// sampler must still be (or wrap) the SWO sampler
	samplerDesc := tracerProviderSamplerDescription(tp)
	t.Logf("sampler description: %s", samplerDesc)
	assert.Contains(t, samplerDesc, "SolarWinds APM Sampler")

	// propagator must still carry W3C Trace Context fields
	fields := otel.GetTextMapPropagator().Fields()
	assert.Contains(t, fields, "traceparent")
	assert.Contains(t, fields, "tracestate")

	// resource must still carry SWO attributes
	resAttrs := resourceAttrsFromProvider(tp)
	t.Logf("resource attrs: %v", resAttrs)
	assert.Equal(t, "apm", resAttrs["sw.data.module"])
	assert.NotEmpty(t, resAttrs["sw.apm.version"])

	// --- meter provider: user's reader must use the user's endpoint ---
	meterDetails := meterProviderReaderDeepDetails(otel.GetMeterProvider())
	var periodicDetail *meterReaderDetail
	for i, d := range meterDetails {
		t.Logf("  reader=%s exporter=%s endpoint=%q", d.ReaderType, d.ExporterType, d.Endpoint)
		if strings.Contains(d.ReaderType, "PeriodicReader") {
			periodicDetail = &meterDetails[i]
		}
	}
	assert.Equal(t, periodicDetail.Endpoint, "196.162.9.776:4317")
	assert.NotContains(t, periodicDetail.Headers, testHeaderToken)
}

// TestStartWithOtelConfNoProviders verifies that when no tracer_provider,
// meter_provider, or logger_provider sections are declared, StartWithOtelConf
// succeeds and all signal providers are noop (not full SDK providers).
func TestStartWithOtelConfNoProviders(t *testing.T) {
	startTestAgent(t, fmt.Sprintf(`file_format: "1.0"
instrumentation/development:
  go:
    solarwinds:
      ServiceKey: "%s"
      RuntimeMetrics: false
`, testServiceKey))

	_, isSdkTP := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	assert.False(t, isSdkTP)

	_, isSdkMP := otel.GetMeterProvider().(*sdkmetric.MeterProvider)
	assert.False(t, isSdkMP)

	_, isNoopLP := otelglobal.GetLoggerProvider().(lognoop.LoggerProvider)
	assert.True(t, isNoopLP)
}

// TestStartWithOtelConfNoProviders verifies that when no tracer_provider,
// meter_provider, or logger_provider sections are declared, StartWithOtelConf
// succeeds and all signal providers are noop (not full SDK providers).
func TestStartWithOtelConfWithAdditionalResourceAttr(t *testing.T) {
	lis := setupLocalListener(t)
	endpoint := lis.Addr().String()

	startTestAgent(t, fmt.Sprintf(`file_format: "1.0"
resource:
  attributes:
    - name: service.name
      value: "gin-otlp-grpc-secure"
    - name: sw.custom.attr
      value: "apm-attr-value"
tracer_provider: {}
meter_provider: {}
instrumentation/development:
  go:
    solarwinds:
      ServiceKey: "%s"
      Collector: "%s"
      RuntimeMetrics: false
`, testServiceKey, endpoint))

	tp, _ := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	// resource must still carry SWO attributes
	resAttrs := resourceAttrsFromProvider(tp)
	t.Logf("resource attrs: %v", resAttrs)
	assert.Equal(t, resAttrs["sw.data.module"], "apm")
	assert.Equal(t, resAttrs["sw.custom.attr"], "apm-attr-value")
	assert.Equal(t, resAttrs["service.name"], "gin-otlp-grpc-secure")
	assert.NotEmpty(t, resAttrs["sw.apm.version"])
}

func TestStartWithOtelConfWithDisableFromYaml(t *testing.T) {
	lis := setupLocalListener(t)
	endpoint := lis.Addr().String()

	startTestAgent(t, fmt.Sprintf(`file_format: "1.0"
disabled: true
resource:
  attributes:
    - name: service.name
      value: "gin-otlp-grpc-secure"
    - name: sw.custom.attr
      value: "apm-attr-value"
tracer_provider: {}
meter_provider: {}
instrumentation/development:
  go:
    solarwinds:
      ServiceKey: "%s"
      Collector: "%s"
      RuntimeMetrics: false
`, testServiceKey, endpoint))

	_, isSdkTP := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	assert.False(t, isSdkTP)
}

// Current behavior is that if two enabled (from swo) and disabled (from otel)
// conflicting, then it's OR operation; e.g. disabled (false) | Enabled (false) -> disabled
func TestStartWithOtelConfWithUserDefinedEnabled(t *testing.T) {
	lis := setupLocalListener(t)
	endpoint := lis.Addr().String()

	startTestAgent(t, fmt.Sprintf(`file_format: "1.0"
disabled: false
resource:
  attributes:
    - name: service.name
      value: "gin-otlp-grpc-secure"
    - name: sw.custom.attr
      value: "apm-attr-value"
tracer_provider: {}
meter_provider: {}
instrumentation/development:
  go:
    solarwinds:
      ServiceKey: "%s"
      Collector: "%s"
      Enabled: false
      RuntimeMetrics: false
`, testServiceKey, endpoint))

	_, isSdkTP := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	assert.False(t, isSdkTP)
}