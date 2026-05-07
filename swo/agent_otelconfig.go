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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/processor"
	"github.com/solarwinds/apm-go/internal/propagator"
	"github.com/solarwinds/apm-go/internal/sampler"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/contrib/otelconf"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/yaml.v3"
)

const (
	otelConfigFileEnv = "OTEL_CONFIG_FILE"
)

type solarwindsOtelConfig struct {
	Collector           *string
	SettingsURL         *string
	ServiceKey          *string
	TrustedPath         *string
	Sampling            *solarwindsSamplingConfig
	PrependDomain       *bool
	HostAlias           *string
	Precision           *int
	SQLSanitize         *int
	TransactionSettings *[]config.TransactionFilter
	Enabled             *bool
	Ec2MetadataTimeout  *int
	DebugLevel          *string
	TriggerTrace        *bool
	Proxy               *string
	ProxyCertPath       *string
	RuntimeMetrics      *bool
	ReportQueryString   *bool
	TokenBucketCap      *float64
	TokenBucketRate     *float64
	TransactionName     *string
}

type solarwindsSamplingConfig struct {
	TracingMode *config.TracingMode `yaml:"TracingMode"`
	SampleRate  *int                `yaml:"SampleRate"`
}

type otelConfigRoot struct {
	InstrumentationDevelopment *otelConfigInstrumentationDevelopment `yaml:"instrumentation/development"`
}

type otelConfigInstrumentationDevelopment struct {
	Go *otelConfigGo `yaml:"go"`
}

type otelConfigGo struct {
	Solarwinds *otelConfigSolarwinds `yaml:"solarwinds"`
}

type otelConfigSolarwinds struct {
	Collector           *string                     `yaml:"Collector"`
	SettingsURL         *string                     `yaml:"SettingsURL"`
	ServiceKey          *string                     `yaml:"ServiceKey"`
	TrustedPath         *string                     `yaml:"TrustedPath"`
	Sampling            *solarwindsSamplingConfig   `yaml:"Sampling"`
	PrependDomain       *bool                       `yaml:"PrependDomain"`
	HostAlias           *string                     `yaml:"HostAlias"`
	Precision           *int                        `yaml:"Precision"`
	SQLSanitize         *int                        `yaml:"SQLSanitize"`
	TransactionSettings *[]config.TransactionFilter `yaml:"TransactionSettings"`
	Enabled             *bool                       `yaml:"Enabled"`
	Ec2MetadataTimeout  *int                        `yaml:"Ec2MetadataTimeout"`
	DebugLevel          *string                     `yaml:"DebugLevel"`
	TriggerTrace        *bool                       `yaml:"TriggerTrace"`
	Proxy               *string                     `yaml:"Proxy"`
	ProxyCertPath       *string                     `yaml:"ProxyCertPath"`
	RuntimeMetrics      *bool                       `yaml:"RuntimeMetrics"`
	ReportQueryString   *bool                       `yaml:"ReportQueryString"`
	TokenBucketCap      *float64                    `yaml:"TokenBucketCap"`
	TokenBucketRate     *float64                    `yaml:"TokenBucketRate"`
	TransactionName     *string                     `yaml:"TransactionName"`
}

func startWithOtelConf(resourceAttrs ...attribute.KeyValue) (func(), error) {
	otelConfigFile := strings.TrimSpace(os.Getenv(otelConfigFileEnv))
	if otelConfigFile == "" {
		log.Debug("OTEL_CONFIG_FILE not set, falling back to legacy startup")
		return startLegacy(resourceAttrs...)
	}
	log.Debug("Detected OTEL_CONFIG_FILE: ", otelConfigFile)

	configBytes, err := os.ReadFile(otelConfigFile)
	if err != nil {
		return func() {}, fmt.Errorf("failed to read %s %q: %w", otelConfigFileEnv, otelConfigFile, err)
	}
	log.Debug("Successfully read OTEL_CONFIG_FILE")

	// Parse instrumentation/development: go: solarwinds:
	solarwindsCfg, err := extractSolarwindsConfigFromOtelYAML(configBytes)
	if err != nil {
		return func() {}, fmt.Errorf("failed to parse SolarWinds configuration in %s %q: %w", otelConfigFileEnv, otelConfigFile, err)
	}
	log.Debug("Extracted SolarWinds configuration from OTEL_CONFIG_FILE")

	otelCfg, err := otelconf.ParseYAML(configBytes)
	if err != nil {
		return func() {}, fmt.Errorf("failed to parse OpenTelemetry configuration in %s %q: %w", otelConfigFileEnv, otelConfigFile, err)
	}
	if otelCfg.Disabled != nil && *otelCfg.Disabled {
		log.Warning("OTEL config file sets disabled=true; otelconf will use noop providers")
	}
	if otelCfg.TracerProvider == nil {
		log.Warning("OTEL config file sets tracer_provider to null/missing; defaults will be used for tracer_provider")
	} else {
		log.Debugf("OTEL config file tracer_provider processors configured: %d", len(otelCfg.TracerProvider.Processors))
	}
	if otelCfg.MeterProvider == nil {
		log.Warning("OTEL config file sets meter_provider to null/missing; defaults will be used for meter_provider")
	}

	// Load the apm-go config from OTEL_CONFIG_FILE values and env vars
	config.Load(buildSolarwindsConfigOptions(solarwindsCfg)...)
	log.Debug("Loaded apm-go configuration from SolarWinds config")

	resrc, err := createResource(resourceAttrs...)
	if err != nil {
		return func() {}, err
	}
	log.Debug("Created resource with ", len(resrc.Attributes()), " attributes")

	// Build default OpenTelemetry configuration with SolarWinds collector endpoint
	// and authorization. NewSDK will read OTEL_CONFIG_FILE and override these
	// defaults with the user's configuration.
	log.Debug("config.GetOtelCollector(): ", config.GetOtelCollector())
	log.Debug("config.GetApiToken() is set: ", strings.TrimSpace(config.GetApiToken()) != "")
	defaultCfg := buildDefaultOtelConfig(config.GetOtelCollector(), config.GetApiToken(), resrc.Attributes())
	log.Debug("Built default OpenTelemetry configuration with collector: ", config.GetOtelCollector())

	o := oboe.NewOboe()
	settingsUpdater, err := oboe.NewSettingsUpdater(o)
	if err != nil {
		log.Error("Failed to create settings updater, ", err)
		return func() {}, err
	}
	log.Debug("Created oboe settings updater")

	ctx := context.Background()
	stopSettingsUpdater := settingsUpdater.Start(ctx)
	log.Debug("Started oboe settings updater with context")

	// Wrap the stopSettingsUpdater to log when it's called
	originalStop := stopSettingsUpdater
	stopSettingsUpdater = func() {
		log.Debug("stopSettingsUpdater() being called")
		originalStop()
		log.Debug("stopSettingsUpdater() completed")
	}

	smplr, err := sampler.NewSampler(o)
	if err != nil {
		return func() { stopSettingsUpdater() }, err
	}
	log.Debug("Created sampler")

	resourceAttrCount := 0
	if defaultCfg.Resource != nil {
		resourceAttrCount = len(defaultCfg.Resource.Attributes)
	}
	log.Debugf(
		"Default OpenTelemetry configuration summary: file_format=%s tracer_provider_set=%t meter_provider_set=%t resource_attrs=%d",
		defaultCfg.FileFormat,
		defaultCfg.TracerProvider != nil,
		defaultCfg.MeterProvider != nil,
		resourceAttrCount,
	)

	mergedCfg := mergeOpenTelemetryConfigWithDefaults(*otelCfg, defaultCfg)
	mergedTraceProcessorCount := 0
	if mergedCfg.TracerProvider != nil {
		mergedTraceProcessorCount = len(mergedCfg.TracerProvider.Processors)
	}
	mergedMeterReaderCount := 0
	if mergedCfg.MeterProvider != nil {
		mergedMeterReaderCount = len(mergedCfg.MeterProvider.Readers)
	}
	log.Debugf(
		"Merged OpenTelemetry configuration summary: file_format=%s tracer_provider_set=%t tracer_processors=%d meter_provider_set=%t meter_readers=%d",
		mergedCfg.FileFormat,
		mergedCfg.TracerProvider != nil,
		mergedTraceProcessorCount,
		mergedCfg.MeterProvider != nil,
		mergedMeterReaderCount,
	)

	log.Debug("mergedCfg.TracerProvider type: ", fmt.Sprintf("%T", mergedCfg.TracerProvider))
	log.Debug("mergedCfg.MeterProvider type: ", fmt.Sprintf("%T", mergedCfg.MeterProvider))

	originalOtelConfigFile, hadOtelConfigFile := os.LookupEnv(otelConfigFileEnv)
	if hadOtelConfigFile {
		if err = os.Unsetenv(otelConfigFileEnv); err != nil {
			stopSettingsUpdater()
			return func() {}, fmt.Errorf("failed to temporarily clear %s: %w", otelConfigFileEnv, err)
		}
		defer func() {
			if restoreErr := os.Setenv(otelConfigFileEnv, originalOtelConfigFile); restoreErr != nil {
				log.Warningf("Failed to restore %s after SDK creation: %v", otelConfigFileEnv, restoreErr)
			}
		}()
		log.Debug("Temporarily cleared OTEL_CONFIG_FILE while creating SDK from merged configuration")
	}

	log.Debug("Creating otelconf SDK from merged configuration")
	sdk, err := otelconf.NewSDK(
		otelconf.WithContext(ctx),
		otelconf.WithOpenTelemetryConfiguration(mergedCfg),
		otelconf.WithTracerProviderOptions(sdktrace.WithSampler(smplr)),
	)
	if err != nil {
		log.Errorf("Failed to create otelconf SDK: %v", err)
		stopSettingsUpdater()
		return func() {}, fmt.Errorf("failed to create otelconf SDK: %w", err)
	}
	log.Debug("Created otelconf SDK")

	tracerProvider := sdk.TracerProvider()
	log.Debugf("NewSDK tracer provider concrete type: %T", tracerProvider)
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(sdk.MeterProvider())
	otelglobal.SetLoggerProvider(sdk.LoggerProvider())
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			sdk.Propagator(),
			&propagator.SolarwindsPropagator{},
		),
	)
	log.Debug("Set global OTEL providers (tracer, meter, logger, propagator)")

	// print out the tracer provider, meter provider detailed information for debugging purposes
	log.Debugf("OTEL TracerProvider after SDK creation: %T", otel.GetTracerProvider())
	log.Debugf("OTEL MeterProvider after SDK creation: %T", otel.GetMeterProvider())
	

	if err = o.RegisterOtelSampleRateMetrics(sdk.MeterProvider()); err != nil {
		_ = sdk.Shutdown(ctx)
		stopSettingsUpdater()
		return func() {}, err
	}
	log.Debug("Registered OTEL sample rate metrics")

	if config.GetRuntimeMetrics() {
		if err = runtime.Start(runtime.WithMeterProvider(sdk.MeterProvider())); err != nil {
			_ = sdk.Shutdown(ctx)
			stopSettingsUpdater()
			return func() {}, err
		}
		log.Debug("Started runtime metrics instrumentation")
	}

	registry, err := metrics.NewOtelRegistry(sdk.MeterProvider())
	if err != nil {
		_ = sdk.Shutdown(ctx)
		stopSettingsUpdater()
		return func() {}, err
	}
	log.Debug("Created metrics registry")

	if tp, ok := tracerProvider.(*sdktrace.TracerProvider); ok {
		maybeRegisterDebugSpanProcessor(tp)
		tp.RegisterSpanProcessor(processor.NewInboundMetricsSpanProcessor(registry))
		log.Debug("Registered inbound metrics span processor")
	} else {
		log.Warningf("Declarative configuration returned a non-SDK tracer provider (%T), inbound metrics span processor was not registered", tracerProvider)
	}

	log.Debug("Declarative OTEL SDK startup completed successfully")
	return func() {
		log.Debug("Shutdown function called, stopping settings updater...")
		stopSettingsUpdater()
		log.Debug("Settings updater stopped")
		
		log.Debug("Shutting down declarative OTEL SDK with context...")
		if shutdownErr := sdk.Shutdown(ctx); shutdownErr != nil {
			log.Error("Failed to shutdown declarative OTEL SDK: ", shutdownErr)
		} else {
			log.Debug("Declarative OTEL SDK shutdown completed successfully")
		}
	}, nil
}

func buildSolarwindsConfigOptions(solarwindsCfg solarwindsOtelConfig) []config.Option {
	opts := []config.Option{}
	if solarwindsCfg.Collector != nil {
		collector := strings.TrimSpace(*solarwindsCfg.Collector)
		opts = append(opts, config.WithCollector(collector))
	}
	if solarwindsCfg.ServiceKey != nil {
		serviceKey := strings.TrimSpace(*solarwindsCfg.ServiceKey)
		opts = append(opts, config.WithServiceKey(serviceKey))
	}
	if solarwindsCfg.SettingsURL != nil {
		settingsURL := strings.TrimSpace(*solarwindsCfg.SettingsURL)
		opts = append(opts, func(c *config.Config) {
			c.SettingsURL = settingsURL
		})
	}
	if solarwindsCfg.TrustedPath != nil {
		trustedPath := strings.TrimSpace(*solarwindsCfg.TrustedPath)
		opts = append(opts, func(c *config.Config) {
			c.TrustedPath = trustedPath
		})
	}
	if solarwindsCfg.Sampling != nil {
		sampling := *solarwindsCfg.Sampling
		opts = append(opts, func(c *config.Config) {
			if c.Sampling == nil {
				c.Sampling = &config.SamplingConfig{}
			}
			if sampling.TracingMode != nil {
				c.Sampling.SetTracingMode(*sampling.TracingMode)
			}
			if sampling.SampleRate != nil {
				c.Sampling.SetSampleRate(*sampling.SampleRate)
			}
		})
	}
	if solarwindsCfg.PrependDomain != nil {
		prependDomain := *solarwindsCfg.PrependDomain
		opts = append(opts, func(c *config.Config) {
			c.PrependDomain = prependDomain
		})
	}
	if solarwindsCfg.HostAlias != nil {
		hostAlias := strings.TrimSpace(*solarwindsCfg.HostAlias)
		opts = append(opts, func(c *config.Config) {
			c.HostAlias = hostAlias
		})
	}
	if solarwindsCfg.Precision != nil {
		precision := *solarwindsCfg.Precision
		opts = append(opts, func(c *config.Config) {
			c.Precision = precision
		})
	}
	if solarwindsCfg.SQLSanitize != nil {
		sqlSanitize := *solarwindsCfg.SQLSanitize
		opts = append(opts, func(c *config.Config) {
			c.SQLSanitize = sqlSanitize
		})
	}
	if solarwindsCfg.TransactionSettings != nil {
		transactionSettings := make([]config.TransactionFilter, len(*solarwindsCfg.TransactionSettings))
		copy(transactionSettings, *solarwindsCfg.TransactionSettings)
		opts = append(opts, func(c *config.Config) {
			c.TransactionSettings = transactionSettings
		})
	}
	if solarwindsCfg.Enabled != nil {
		enabled := *solarwindsCfg.Enabled
		opts = append(opts, func(c *config.Config) {
			c.Enabled = enabled
		})
	}
	if solarwindsCfg.Ec2MetadataTimeout != nil {
		ec2MetadataTimeout := *solarwindsCfg.Ec2MetadataTimeout
		opts = append(opts, func(c *config.Config) {
			c.Ec2MetadataTimeout = ec2MetadataTimeout
		})
	}
	if solarwindsCfg.DebugLevel != nil {
		debugLevel := strings.TrimSpace(*solarwindsCfg.DebugLevel)
		opts = append(opts, func(c *config.Config) {
			c.DebugLevel = debugLevel
		})
	}
	if solarwindsCfg.TriggerTrace != nil {
		triggerTrace := *solarwindsCfg.TriggerTrace
		opts = append(opts, func(c *config.Config) {
			c.TriggerTrace = triggerTrace
		})
	}
	if solarwindsCfg.Proxy != nil {
		proxy := strings.TrimSpace(*solarwindsCfg.Proxy)
		opts = append(opts, func(c *config.Config) {
			c.Proxy = proxy
		})
	}
	if solarwindsCfg.ProxyCertPath != nil {
		proxyCertPath := strings.TrimSpace(*solarwindsCfg.ProxyCertPath)
		opts = append(opts, func(c *config.Config) {
			c.ProxyCertPath = proxyCertPath
		})
	}
	if solarwindsCfg.RuntimeMetrics != nil {
		runtimeMetrics := *solarwindsCfg.RuntimeMetrics
		opts = append(opts, config.WithRuntimeMetrics(runtimeMetrics))
	}
	if solarwindsCfg.ReportQueryString != nil {
		reportQueryString := *solarwindsCfg.ReportQueryString
		opts = append(opts, func(c *config.Config) {
			c.ReportQueryString = reportQueryString
		})
	}
	if solarwindsCfg.TokenBucketCap != nil {
		tokenBucketCap := *solarwindsCfg.TokenBucketCap
		opts = append(opts, func(c *config.Config) {
			c.TokenBucketCap = tokenBucketCap
		})
	}
	if solarwindsCfg.TokenBucketRate != nil {
		tokenBucketRate := *solarwindsCfg.TokenBucketRate
		opts = append(opts, func(c *config.Config) {
			c.TokenBucketRate = tokenBucketRate
		})
	}
	if solarwindsCfg.TransactionName != nil {
		transactionName := strings.TrimSpace(*solarwindsCfg.TransactionName)
		opts = append(opts, func(c *config.Config) {
			c.TransactionName = transactionName
		})
	}
	return opts
}

func extractSolarwindsConfigFromOtelYAML(configBytes []byte) (solarwindsOtelConfig, error) {
	root := otelConfigRoot{}
	if err := yaml.Unmarshal(configBytes, &root); err != nil {
		return solarwindsOtelConfig{}, err
	}
	if root.InstrumentationDevelopment == nil ||
		root.InstrumentationDevelopment.Go == nil ||
		root.InstrumentationDevelopment.Go.Solarwinds == nil {
		return solarwindsOtelConfig{}, nil
	}

	solarwindsCfg := root.InstrumentationDevelopment.Go.Solarwinds
	return solarwindsOtelConfig{
		Collector:           solarwindsCfg.Collector,
		SettingsURL:         solarwindsCfg.SettingsURL,
		ServiceKey:          solarwindsCfg.ServiceKey,
		TrustedPath:         solarwindsCfg.TrustedPath,
		Sampling:            solarwindsCfg.Sampling,
		PrependDomain:       solarwindsCfg.PrependDomain,
		HostAlias:           solarwindsCfg.HostAlias,
		Precision:           solarwindsCfg.Precision,
		SQLSanitize:         solarwindsCfg.SQLSanitize,
		TransactionSettings: solarwindsCfg.TransactionSettings,
		Enabled:             solarwindsCfg.Enabled,
		Ec2MetadataTimeout:  solarwindsCfg.Ec2MetadataTimeout,
		DebugLevel:          solarwindsCfg.DebugLevel,
		TriggerTrace:        solarwindsCfg.TriggerTrace,
		Proxy:               solarwindsCfg.Proxy,
		ProxyCertPath:       solarwindsCfg.ProxyCertPath,
		RuntimeMetrics:      solarwindsCfg.RuntimeMetrics,
		ReportQueryString:   solarwindsCfg.ReportQueryString,
		TokenBucketCap:      solarwindsCfg.TokenBucketCap,
		TokenBucketRate:     solarwindsCfg.TokenBucketRate,
		TransactionName:     solarwindsCfg.TransactionName,
	}, nil
}

func buildDefaultOtelConfig(collectorEndpoint, apiToken string, resourceAttrs []attribute.KeyValue) otelconf.OpenTelemetryConfiguration {
	cfg := otelconf.OpenTelemetryConfiguration{
		FileFormat: "1.0",
	}

	if len(resourceAttrs) > 0 {
		res := &otelconf.Resource{}
		for _, attr := range resourceAttrs {
			converted, ok := resourceAttributeToNameValue(attr)
			if ok {
				res.Attributes = append(res.Attributes, converted)
			}
		}
		cfg.Resource = res
	}

	if collectorEndpoint == "" {
		return cfg
	}

	var headers []otelconf.NameStringValuePair
	if strings.TrimSpace(apiToken) != "" {
		headers = []otelconf.NameStringValuePair{
			{
				Name:  "Authorization",
				Value: toNameStringValue(fmt.Sprintf("Bearer %s", apiToken)),
			},
		}
	}

	cfg.TracerProvider = &otelconf.TracerProvider{
		Processors: []otelconf.SpanProcessor{
			{
				Batch: &otelconf.BatchSpanProcessor{
					Exporter: otelconf.SpanExporter{
						OTLPGrpc: &otelconf.OTLPGrpcExporter{
							Endpoint: otelconf.OTLPGrpcExporterEndpoint(stringPtr(collectorEndpoint)),
							Headers:  headers,
						},
					},
				},
			},
		},
	}

	cfg.MeterProvider = &otelconf.MeterProvider{
		Readers: []otelconf.MetricReader{
			{
				Periodic: &otelconf.PeriodicMetricReader{
					Exporter: otelconf.PushMetricExporter{
						OTLPGrpc: &otelconf.OTLPGrpcMetricExporter{
							Endpoint: otelconf.OTLPGrpcMetricExporterEndpoint(stringPtr(collectorEndpoint)),
							Headers:  headers,
						},
					},
				},
			},
		},
	}

	return cfg
}

func mergeOpenTelemetryConfigWithDefaults(fileCfg, defaultCfg otelconf.OpenTelemetryConfiguration) otelconf.OpenTelemetryConfiguration {
	mergedCfg := fileCfg

	if strings.TrimSpace(mergedCfg.FileFormat) == "" {
		mergedCfg.FileFormat = defaultCfg.FileFormat
	}
	if mergedCfg.Resource == nil {
		mergedCfg.Resource = defaultCfg.Resource
	}

	if mergedCfg.TracerProvider == nil {
		mergedCfg.TracerProvider = defaultCfg.TracerProvider
	} else if defaultCfg.TracerProvider != nil && len(mergedCfg.TracerProvider.Processors) == 0 {
		mergedCfg.TracerProvider.Processors = defaultCfg.TracerProvider.Processors
	}

	if mergedCfg.MeterProvider == nil {
		mergedCfg.MeterProvider = defaultCfg.MeterProvider
	} else if defaultCfg.MeterProvider != nil && len(mergedCfg.MeterProvider.Readers) == 0 {
		mergedCfg.MeterProvider.Readers = defaultCfg.MeterProvider.Readers
	}

	return mergedCfg
}

func toNameStringValue(value string) otelconf.NameStringValuePairValue {
	return otelconf.NameStringValuePairValue(stringPtr(value))
}

func stringPtr(value string) *string {
	return &value
}

func resourceAttributeToNameValue(kv attribute.KeyValue) (otelconf.AttributeNameValue, bool) {
	attr := otelconf.AttributeNameValue{Name: string(kv.Key)}
	switch kv.Value.Type() {
	case attribute.BOOL:
		attr.Value = kv.Value.AsBool()
	case attribute.INT64:
		attr.Value = kv.Value.AsInt64()
	case attribute.FLOAT64:
		attr.Value = kv.Value.AsFloat64()
	case attribute.STRING:
		attr.Value = kv.Value.AsString()
	default:
		return otelconf.AttributeNameValue{}, false
	}
	return attr, true
}

type debugSpanProcessor struct{}

func maybeRegisterDebugSpanProcessor(tp *sdktrace.TracerProvider) {
	if log.Level() != log.DEBUG {
		return
	}

	tp.RegisterSpanProcessor(debugSpanProcessor{})
	log.Debug("Registered debug span processor")
}

func (debugSpanProcessor) OnStart(_ context.Context, span sdktrace.ReadWriteSpan) {
	sc := span.SpanContext()
	log.Debugf(
		"OTEL span started: name=%q trace_id=%s span_id=%s sampled=%t",
		span.Name(),
		sc.TraceID().String(),
		sc.SpanID().String(),
		sc.IsSampled(),
	)
}

func (debugSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	sc := span.SpanContext()
	log.Debugf(
		"OTEL span ended: name=%q trace_id=%s span_id=%s sampled=%t",
		span.Name(),
		sc.TraceID().String(),
		sc.SpanID().String(),
		sc.IsSampled(),
	)
}

func (debugSpanProcessor) Shutdown(context.Context) error {
	return nil
}

func (debugSpanProcessor) ForceFlush(context.Context) error {
	return nil
}
