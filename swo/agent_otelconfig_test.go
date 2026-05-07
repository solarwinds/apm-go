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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/otelconf"
	"go.opentelemetry.io/otel/attribute"
)

func getHeaderValue(headers []otelconf.NameStringValuePair, key string) string {
	for _, h := range headers {
		if !strings.EqualFold(strings.TrimSpace(h.Name), strings.TrimSpace(key)) {
			continue
		}
		if h.Value == nil {
			return ""
		}
		return *h.Value
	}
	return ""
}

func TestExtractSolarwindsConfigFromOtelYAML(t *testing.T) {
	configYAML := "file_format: \"1.0\"\n" +
		"instrumentation/development:\n" +
		"  go:\n" +
		"    solarwinds:\n" +
		"      ServiceKey: \"token:service\"\n" +
		"      Collector: \"apm.collector.na-01.cloud.solarwinds.com:443\"\n" +
		"      RuntimeMetrics: false\n"

	solarwindsCfg, err := extractSolarwindsConfigFromOtelYAML([]byte(configYAML))
	require.NoError(t, err)
	require.NotNil(t, solarwindsCfg.ServiceKey)
	require.NotNil(t, solarwindsCfg.Collector)
	require.NotNil(t, solarwindsCfg.RuntimeMetrics)

	assert.Equal(t, "token:service", *solarwindsCfg.ServiceKey)
	assert.Equal(t, "apm.collector.na-01.cloud.solarwinds.com:443", *solarwindsCfg.Collector)
	assert.False(t, *solarwindsCfg.RuntimeMetrics)
}

// this is working yaml for current setup
func TestSampleYaml(t *testing.T) {
	configYAML := "file_format: \"1.0\"\n" +
		"resource:\n" +
		"  attributes:\n" +
		"    - name: service.name\n" +
		"      value: \"golang-xuan-test\"\n" +
		"    - name: service.version\n" +
		"      value: \"1.0.0\"\n" +
		"    - name: deployment.environment\n" +
		"      value: \"staging\"\n" +
		"instrumentation/development:\n" +
		"  go:\n" +
		"    solarwinds:\n" +
		"      ServiceKey: ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:test-service\n" +
		"      Collector:  apm.collector.na-01.st-ssp.solarwinds.com:443\n" +
		"      RuntimeMetrics: true\n" +
		"      DebugLevel: \"DEBUG\"\n"

	solarwindsCfg, err := extractSolarwindsConfigFromOtelYAML([]byte(configYAML))
	require.NoError(t, err)
	require.NotNil(t, solarwindsCfg.ServiceKey)
	require.NotNil(t, solarwindsCfg.Collector)
	require.NotNil(t, solarwindsCfg.RuntimeMetrics)
	require.NotNil(t, solarwindsCfg.DebugLevel)

	assert.Equal(t, "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:test-service", *solarwindsCfg.ServiceKey)
	assert.Equal(t, "apm.collector.na-01.st-ssp.solarwinds.com:443", *solarwindsCfg.Collector)
	assert.True(t, *solarwindsCfg.RuntimeMetrics)
	assert.Equal(t, "DEBUG", *solarwindsCfg.DebugLevel)
}

func TestExtractSolarwindsConfigFromOtelYAML_MissingSolarwindsSection(t *testing.T) {
	configYAML := "file_format: \"1.0\"\n" +
		"tracer_provider:\n" +
		"  processors:\n" +
		"    - batch:\n" +
		"        exporter:\n" +
		"          otlp_grpc:\n" +
		"            endpoint: \"http://localhost:4317\"\n"

	solarwindsCfg, err := extractSolarwindsConfigFromOtelYAML([]byte(configYAML))
	require.NoError(t, err)
	assert.Nil(t, solarwindsCfg.ServiceKey)
	assert.Nil(t, solarwindsCfg.Collector)
	assert.Nil(t, solarwindsCfg.RuntimeMetrics)
}

func TestExtractSolarwindsConfigFromOtelYAML_InvalidYAML(t *testing.T) {
	_, err := extractSolarwindsConfigFromOtelYAML([]byte("file_format: ["))
	require.Error(t, err)
}

func TestBuildSolarwindsConfigOptions_MapsExtendedFields(t *testing.T) {
	trustedPath := filepath.Join(t.TempDir(), "trusted.pem")
	require.NoError(t, os.WriteFile(trustedPath, []byte("test-cert"), 0o600))

	serviceKey := "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:test-service"
	collector := "custom.collector.na-01.cloud.solarwinds.com:443"
	settingsURL := "https://settings.example.com"
	hostAlias := "my-host"
	debugLevel := "error"
	proxy := "http://127.0.0.1:8080"
	proxyCertPath := trustedPath
	prependDomain := true
	precision := 4
	sqlSanitize := 2
	ec2MetadataTimeout := 750
	triggerTrace := false
	runtimeMetrics := false
	reportQueryString := false
	tokenBucketCap := 3.5
	tokenBucketRate := 0.8
	tracingMode := config.DisabledTracingMode
	sampleRate := 1234

	solarwindsCfg := solarwindsOtelConfig{
		Collector:          &collector,
		SettingsURL:        &settingsURL,
		ServiceKey:         &serviceKey,
		TrustedPath:        &trustedPath,
		Sampling:           &solarwindsSamplingConfig{TracingMode: &tracingMode, SampleRate: &sampleRate},
		PrependDomain:      &prependDomain,
		HostAlias:          &hostAlias,
		Precision:          &precision,
		SQLSanitize:        &sqlSanitize,
		Ec2MetadataTimeout: &ec2MetadataTimeout,
		DebugLevel:         &debugLevel,
		TriggerTrace:       &triggerTrace,
		Proxy:              &proxy,
		ProxyCertPath:      &proxyCertPath,
		RuntimeMetrics:     &runtimeMetrics,
		ReportQueryString:  &reportQueryString,
		TokenBucketCap:     &tokenBucketCap,
		TokenBucketRate:    &tokenBucketRate,
	}

	cfg := config.NewConfig(buildSolarwindsConfigOptions(solarwindsCfg)...)
	require.True(t, cfg.GetEnabled())

	assert.Equal(t, collector, cfg.GetCollector())
	assert.Equal(t, settingsURL, cfg.GetSettingsURL())
	assert.Equal(t, serviceKey, cfg.GetServiceKey())
	assert.Equal(t, trustedPath, cfg.GetTrustedPath())
	assert.Equal(t, config.DisabledTracingMode, cfg.GetTracingMode())
	assert.Equal(t, sampleRate, cfg.GetSampleRate())
	assert.True(t, cfg.GetPrependDomain())
	assert.Equal(t, hostAlias, cfg.GetHostAlias())
	assert.Equal(t, precision, cfg.GetPrecision())
	assert.Equal(t, sqlSanitize, cfg.GetSQLSanitize())
	assert.Equal(t, ec2MetadataTimeout, cfg.GetEc2MetadataTimeout())
	assert.Equal(t, debugLevel, cfg.GetDebugLevel())
	assert.False(t, cfg.GetTriggerTrace())
	assert.Equal(t, proxy, cfg.GetProxy())
	assert.Equal(t, proxyCertPath, cfg.GetProxyCertPath())
	assert.False(t, cfg.GetRuntimeMetrics())
	assert.False(t, cfg.GetReportQueryString())
	assert.Equal(t, tokenBucketCap, cfg.GetTokenBucketCap())
	assert.Equal(t, tokenBucketRate, cfg.GetTokenBucketRate())
}

func TestBuildSolarwindsConfigOptions_TrimsWhitespaceAndCopiesTransactionFilters(t *testing.T) {
	collector := "  custom.collector.na-01.cloud.solarwinds.com:443  "
	serviceKey := "  ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:test-service  "
	hostAlias := "  my-host  "
	settingsURL := "  https://settings.example.com  "
	debugLevel := "  warn  "
	transactionFilters := []config.TransactionFilter{
		{
			Type:    config.URL,
			RegEx:   "/checkout",
			Tracing: config.DisabledTracingMode,
		},
	}

	solarwindsCfg := solarwindsOtelConfig{
		Collector:           &collector,
		ServiceKey:          &serviceKey,
		HostAlias:           &hostAlias,
		SettingsURL:         &settingsURL,
		DebugLevel:          &debugLevel,
		TransactionSettings: &transactionFilters,
	}

	opts := buildSolarwindsConfigOptions(solarwindsCfg)
	transactionFilters[0].RegEx = "/mutated"

	cfg := config.NewConfig(opts...)
	assert.Equal(t, "custom.collector.na-01.cloud.solarwinds.com:443", cfg.GetCollector())
	assert.Equal(t, "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:test-service", cfg.GetServiceKey())
	assert.Equal(t, "my-host", cfg.GetHostAlias())
	assert.Equal(t, "https://settings.example.com", cfg.GetSettingsURL())
	assert.Equal(t, "warn", cfg.GetDebugLevel())

	filters := cfg.GetTransactionFiltering()
	require.Len(t, filters, 1)
	assert.Equal(t, "/checkout", filters[0].RegEx)
}

func TestBuildDefaultOtelConfig_SetsEndpointAndHeaders(t *testing.T) {
	collectorEndpoint := "https://otel.collector.na-01.cloud.solarwinds.com:443"
	apiToken := "test-token"

	cfg := buildDefaultOtelConfig(collectorEndpoint, apiToken, nil)

	assert.Equal(t, "1.0", cfg.FileFormat)

	require.NotNil(t, cfg.TracerProvider)
	require.Len(t, cfg.TracerProvider.Processors, 1)
	traceExporter := cfg.TracerProvider.Processors[0].Batch.Exporter.OTLPGrpc
	require.NotNil(t, traceExporter)
	require.NotNil(t, traceExporter.Endpoint)
	assert.Equal(t, collectorEndpoint, *traceExporter.Endpoint)
	assert.Equal(t, "Bearer test-token", getHeaderValue(traceExporter.Headers, "authorization"))

	require.NotNil(t, cfg.MeterProvider)
	require.Len(t, cfg.MeterProvider.Readers, 1)
	metricExporter := cfg.MeterProvider.Readers[0].Periodic.Exporter.OTLPGrpc
	require.NotNil(t, metricExporter)
	require.NotNil(t, metricExporter.Endpoint)
	assert.Equal(t, collectorEndpoint, *metricExporter.Endpoint)
	assert.Equal(t, "Bearer test-token", getHeaderValue(metricExporter.Headers, "authorization"))

	assert.Nil(t, cfg.LoggerProvider)
}

func TestBuildDefaultOtelConfig_EmptyCollector(t *testing.T) {
	cfg := buildDefaultOtelConfig("", "test-token", nil)

	assert.Equal(t, "1.0", cfg.FileFormat)
	assert.Nil(t, cfg.TracerProvider)
	assert.Nil(t, cfg.MeterProvider)
	assert.Nil(t, cfg.LoggerProvider)
}

func TestBuildDefaultOtelConfig_NoApiToken(t *testing.T) {
	collectorEndpoint := "https://otel.collector.na-01.cloud.solarwinds.com:443"
	cfg := buildDefaultOtelConfig(collectorEndpoint, "", nil)

	traceExporter := cfg.TracerProvider.Processors[0].Batch.Exporter.OTLPGrpc
	require.NotNil(t, traceExporter)
	assert.Len(t, traceExporter.Headers, 0)

	metricExporter := cfg.MeterProvider.Readers[0].Periodic.Exporter.OTLPGrpc
	require.NotNil(t, metricExporter)
	assert.Len(t, metricExporter.Headers, 0)
}

func TestBuildDefaultOtelConfig_WhitespaceApiToken(t *testing.T) {
	collectorEndpoint := "https://otel.collector.na-01.cloud.solarwinds.com:443"
	cfg := buildDefaultOtelConfig(collectorEndpoint, "   ", nil)

	traceExporter := cfg.TracerProvider.Processors[0].Batch.Exporter.OTLPGrpc
	require.NotNil(t, traceExporter)
	assert.Len(t, traceExporter.Headers, 0)

	metricExporter := cfg.MeterProvider.Readers[0].Periodic.Exporter.OTLPGrpc
	require.NotNil(t, metricExporter)
	assert.Len(t, metricExporter.Headers, 0)
}

func TestBuildDefaultOtelConfig_IncludesResourceAttributes(t *testing.T) {
	attrs := []attribute.KeyValue{
		attribute.String("sw.data.module", "apm"),
		attribute.String("sw.apm.version", "1.0.0"),
		attribute.Int64("uams.client.id", 42),
		attribute.Bool("custom.flag", true),
	}

	cfg := buildDefaultOtelConfig("https://collector:443", "token", attrs)

	require.NotNil(t, cfg.Resource)
	assert.Len(t, cfg.Resource.Attributes, 4)

	attrMap := make(map[string]interface{})
	for _, a := range cfg.Resource.Attributes {
		attrMap[a.Name] = a.Value
	}
	assert.Equal(t, "apm", attrMap["sw.data.module"])
	assert.Equal(t, "1.0.0", attrMap["sw.apm.version"])
	assert.Equal(t, int64(42), attrMap["uams.client.id"])
	assert.Equal(t, true, attrMap["custom.flag"])
}

func TestBuildDefaultOtelConfig_SkipsUnsupportedResourceAttributes(t *testing.T) {
	attrs := []attribute.KeyValue{
		attribute.StringSlice("skip.me", []string{"a", "b"}),
		attribute.String("keep.me", "ok"),
	}

	cfg := buildDefaultOtelConfig("https://collector:443", "token", attrs)
	require.NotNil(t, cfg.Resource)
	require.Len(t, cfg.Resource.Attributes, 1)
	assert.Equal(t, "keep.me", cfg.Resource.Attributes[0].Name)
	assert.Equal(t, "ok", cfg.Resource.Attributes[0].Value)
}

func TestMergeOpenTelemetryConfigWithDefaults_FillsNilProviders(t *testing.T) {
	fileCfgYAML := "file_format: \"1.0\"\n" +
		"tracer_provider:\n" +
		"meter_provider:\n"

	fileCfg, err := otelconf.ParseYAML([]byte(fileCfgYAML))
	require.NoError(t, err)
	require.Nil(t, fileCfg.TracerProvider)
	require.Nil(t, fileCfg.MeterProvider)

	defaultCfg := buildDefaultOtelConfig(
		"https://otel.collector.na-01.cloud.solarwinds.com:443",
		"token",
		[]attribute.KeyValue{attribute.String("service.name", "from-default")},
	)

	mergedCfg := mergeOpenTelemetryConfigWithDefaults(*fileCfg, defaultCfg)
	require.NotNil(t, mergedCfg.TracerProvider)
	require.NotNil(t, mergedCfg.MeterProvider)
	require.NotNil(t, mergedCfg.Resource)
	assert.Len(t, mergedCfg.TracerProvider.Processors, 1)
	assert.Len(t, mergedCfg.MeterProvider.Readers, 1)
	assert.Len(t, mergedCfg.Resource.Attributes, 1)
	assert.Equal(t, "service.name", mergedCfg.Resource.Attributes[0].Name)
	assert.Equal(t, "from-default", mergedCfg.Resource.Attributes[0].Value)
}

func TestMergeOpenTelemetryConfigWithDefaults_FillsEmptyProviderBlocks(t *testing.T) {
	fileCfgYAML := "file_format: \"1.0\"\n" +
		"resource:\n" +
		"  attributes:\n" +
		"    - name: service.name\n" +
		"      value: from-file\n" +
		"tracer_provider: {}\n" +
		"meter_provider: {}\n"

	fileCfg, err := otelconf.ParseYAML([]byte(fileCfgYAML))
	require.NoError(t, err)
	require.NotNil(t, fileCfg.TracerProvider)
	require.NotNil(t, fileCfg.MeterProvider)
	require.Len(t, fileCfg.TracerProvider.Processors, 0)
	require.Len(t, fileCfg.MeterProvider.Readers, 0)

	defaultCfg := buildDefaultOtelConfig(
		"https://otel.collector.na-01.cloud.solarwinds.com:443",
		"token",
		[]attribute.KeyValue{attribute.String("service.name", "from-default")},
	)

	mergedCfg := mergeOpenTelemetryConfigWithDefaults(*fileCfg, defaultCfg)
	require.NotNil(t, mergedCfg.TracerProvider)
	require.NotNil(t, mergedCfg.MeterProvider)
	assert.Len(t, mergedCfg.TracerProvider.Processors, 1)
	assert.Len(t, mergedCfg.MeterProvider.Readers, 1)

	require.NotNil(t, mergedCfg.Resource)
	require.Len(t, mergedCfg.Resource.Attributes, 1)
	assert.Equal(t, "service.name", mergedCfg.Resource.Attributes[0].Name)
	assert.Equal(t, "from-file", mergedCfg.Resource.Attributes[0].Value)
}

func TestGetTraceExporterEndpointFromConfig_OTLPGrpc(t *testing.T) {
	cfg := buildDefaultOtelConfig("https://collector:443", "token", nil)

	endpoint, ok := getTraceExporterEndpointFromConfig(cfg)
	require.True(t, ok)
	assert.Equal(t, "https://collector:443", endpoint)
}

func TestGetMetricExporterEndpointFromConfig_OTLPGrpc(t *testing.T) {
	cfg := buildDefaultOtelConfig("https://collector:443", "token", nil)

	endpoint, ok := getMetricExporterEndpointFromConfig(cfg)
	require.True(t, ok)
	assert.Equal(t, "https://collector:443", endpoint)
}

func TestGetTraceExporterEndpointFromConfig_None(t *testing.T) {
	endpoint, ok := getTraceExporterEndpointFromConfig(otelconf.OpenTelemetryConfiguration{})
	assert.False(t, ok)
	assert.Equal(t, "", endpoint)
}

func TestGetMetricExporterEndpointFromConfig_None(t *testing.T) {
	endpoint, ok := getMetricExporterEndpointFromConfig(otelconf.OpenTelemetryConfiguration{})
	assert.False(t, ok)
	assert.Equal(t, "", endpoint)
}

func TestResourceAttributeToNameValue_UnsupportedType(t *testing.T) {
	_, ok := resourceAttributeToNameValue(attribute.StringSlice("unsupported", []string{"a", "b"}))
	assert.False(t, ok)
}

func TestComprehensiveDeclarativeConfig_ExtractsAndBuildsSolarwindsConfig(t *testing.T) {
	configYAML := "file_format: \"1.0\"\n" +
		"disabled: false\n" +
		"log_level: info\n" +
		"resource:\n" +
		"  attributes:\n" +
		"    - name: service.name\n" +
		"      value: unknown_service\n" +
		"propagator:\n" +
		"  composite:\n" +
		"    - tracecontext:\n" +
		"    - baggage:\n" +
		"tracer_provider:\n" +
		"  processors:\n" +
		"    - batch:\n" +
		"        schedule_delay: 5000\n" +
		"        export_timeout: 30000\n" +
		"        max_queue_size: 2048\n" +
		"        max_export_batch_size: 512\n" +
		"        exporter:\n" +
		"          otlp_http:\n" +
		"            endpoint: http://localhost:4318/v1/traces\n" +
		"            compression: gzip\n" +
		"            timeout: 10000\n" +
		"meter_provider:\n" +
		"  readers:\n" +
		"    - periodic:\n" +
		"        interval: 60000\n" +
		"        timeout: 30000\n" +
		"        exporter:\n" +
		"          otlp_http:\n" +
		"            endpoint: http://localhost:4318/v1/metrics\n" +
		"            compression: gzip\n" +
		"            timeout: 10000\n" +
		"logger_provider:\n" +
		"  processors:\n" +
		"    - batch:\n" +
		"        schedule_delay: 1000\n" +
		"        export_timeout: 30000\n" +
		"        max_queue_size: 2048\n" +
		"        max_export_batch_size: 512\n" +
		"        exporter:\n" +
		"          otlp_http:\n" +
		"            endpoint: http://localhost:4318/v1/logs\n" +
		"            compression: gzip\n" +
		"            timeout: 10000\n" +
		"instrumentation/development:\n" +
		"  go:\n" +
		"    solarwinds:\n" +
		"      ServiceKey: \"ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:service\"\n" +
		"      Collector: \"apm.collector.na-01.cloud.solarwinds.com:443\"\n" +
		"      RuntimeMetrics: false\n" +
		"      HostAlias: \"my-host\"\n"

	solarwindsCfg, err := extractSolarwindsConfigFromOtelYAML([]byte(configYAML))
	require.NoError(t, err)
	require.NotNil(t, solarwindsCfg.ServiceKey)
	require.NotNil(t, solarwindsCfg.Collector)
	require.NotNil(t, solarwindsCfg.RuntimeMetrics)
	require.NotNil(t, solarwindsCfg.HostAlias)

	cfg := config.NewConfig(buildSolarwindsConfigOptions(solarwindsCfg)...)
	assert.Equal(t, "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:service", cfg.GetServiceKey())
	assert.Equal(t, "apm.collector.na-01.cloud.solarwinds.com:443", cfg.GetCollector())
	assert.Equal(t, "my-host", cfg.GetHostAlias())
	assert.False(t, cfg.GetRuntimeMetrics())

	// Build default config — these defaults would be overridden by the user's YAML
	// when NewSDK reads OTEL_CONFIG_FILE
	defaultCfg := buildDefaultOtelConfig(
		"https://otel.collector.na-01.cloud.solarwinds.com:443",
		"token",
		nil,
	)
	require.NotNil(t, defaultCfg.TracerProvider)
	require.NotNil(t, defaultCfg.MeterProvider)
	assert.Nil(t, defaultCfg.LoggerProvider)

	traceExporter := defaultCfg.TracerProvider.Processors[0].Batch.Exporter.OTLPGrpc
	require.NotNil(t, traceExporter)
	require.NotNil(t, traceExporter.Endpoint)
	assert.Equal(t, "https://otel.collector.na-01.cloud.solarwinds.com:443", *traceExporter.Endpoint)
	assert.Equal(t, "Bearer token", getHeaderValue(traceExporter.Headers, "authorization"))

	metricExporter := defaultCfg.MeterProvider.Readers[0].Periodic.Exporter.OTLPGrpc
	require.NotNil(t, metricExporter)
	require.NotNil(t, metricExporter.Endpoint)
	assert.Equal(t, "https://otel.collector.na-01.cloud.solarwinds.com:443", *metricExporter.Endpoint)
	assert.Equal(t, "Bearer token", getHeaderValue(metricExporter.Headers, "authorization"))
}


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

package swo

import (
	"context"
	"os"

	stdlog "log"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/otelsetup"
	"github.com/solarwinds/apm-go/internal/processor"
	"github.com/solarwinds/apm-go/internal/propagator"
	"github.com/solarwinds/apm-go/internal/reporter"
	"github.com/solarwinds/apm-go/internal/sampler"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Start bootstraps otel requirements and starts the agent. The given `resourceAttrs` are added to the otel
// `resource.Resource` that is supplied to the otel `TracerProvider`
func Start(resourceAttrs ...attribute.KeyValue) (func(), error) {
	if !config.GetEnabled() {
		log.Info("SolarWinds Observability APM agent is disabled, skipping startup.")
		return func() {}, nil
	}

	if os.Getenv("OTEL_CONFIG_FILE") != "" {
		log.Debug("OTEL_CONFIG_FILE environment variable detected, starting with otel configuration")
		return startWithOtelConf(resourceAttrs...)
	}

	return startLegacy(resourceAttrs...)
}

func startLegacy(resourceAttrs ...attribute.KeyValue) (func(), error) {
	resrc, err := createResource(resourceAttrs...)
	if err != nil {
		return func() {
			// return a no-op func so that we don't cause a nil-deref for the end-user
		}, err
	}
	o := oboe.NewOboe()

	settingsUpdater, err := oboe.NewSettingsUpdater(o)
	if err != nil {
		log.Error("Failed to create settings updater, ", err)
		return func() {}, err
	}

	ctx := context.Background()
	stopSettingsUpdater := settingsUpdater.Start(ctx)

	exprtr, err := otelsetup.NewSpanExporter(ctx)
	if err != nil {
		log.Error("Failed to configure span exporter, ", err)
		return func() { stopSettingsUpdater() }, err
	}

	metricsPublisher := reporter.NewMetricsPublisher()
	err = metricsPublisher.ConfigureAndStart(ctx, o, resrc)
	if err != nil {
		log.Error("Failed to configure and start metrics publisher, ", err)
		return func() { stopSettingsUpdater() }, err
	}

	smplr, err := sampler.NewSampler(o)
	if err != nil {
		return func() { stopSettingsUpdater() }, err
	}

	proc := processor.NewInboundMetricsSpanProcessor(metricsPublisher.GetMetricsRegistry())
	prop := propagation.NewCompositeTextMapPropagator(
		&propagation.TraceContext{},
		&propagation.Baggage{},
		&propagator.SolarwindsPropagator{},
	)
	otel.SetTextMapPropagator(prop)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exprtr),
		sdktrace.WithResource(resrc),
		sdktrace.WithSampler(smplr),
		sdktrace.WithSpanProcessor(proc),
	)
	otel.SetTracerProvider(tp)

	return func() {
		stopSettingsUpdater()

		err := metricsPublisher.Shutdown()
		if err != nil {
			log.Error("Failed to shutdown metrics publisher: ", err)
		}
		if err = tp.Shutdown(ctx); err != nil {
			stdlog.Fatal(err)
		}
	}, nil
}
