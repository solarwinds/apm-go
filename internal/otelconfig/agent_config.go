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
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/sworesource"
	"github.com/solarwinds/apm-go/internal/constants"
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
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// StartWithOtelConf starts the agent using the OpenTelemetry configuration file
// specified by OtelConfigFileEnv.
func StartWithOtelConf(resourceAttrs ...attribute.KeyValue) (func(), error) {
	otelConfigFile := strings.TrimSpace(os.Getenv(constants.OtelConfigFileEnv))

	configBytes, err := os.ReadFile(otelConfigFile)
	if err != nil {
		return func() {}, fmt.Errorf("failed to read %s %q: %w", constants.OtelConfigFileEnv, otelConfigFile, err)
	}

	otelCfg, err := otelconf.ParseYAML(configBytes)
	if err != nil {
		return func() {}, fmt.Errorf("failed to parse OpenTelemetry configuration in %s %q: %w", constants.OtelConfigFileEnv, otelConfigFile, err)
	}

	if otelCfg.Disabled != nil && *otelCfg.Disabled {
		log.Warning("OTEL config file sets disabled=true; otelconf will use noop providers")
		return func() {}, nil
	}

	// Parse the SolarWinds-specific configuration from the same YAML file, so that we can apply it before creating the SDK.
	// This allows users to specify all configuration in a single file if they choose to.
	solarwindsCfg, err := extractSolarwindsConfigFromOtelYAML(configBytes)
	if err != nil {
		return func() {}, fmt.Errorf("failed to parse SolarWinds configuration in %s %q: %w", constants.OtelConfigFileEnv, otelConfigFile, err)
	}

	log.Debug("config.Load from otelconf")
	config.Load(buildSolarwindsConfigOptions(solarwindsCfg)...)
	log.Debug("config.Load from otelconf end")
	if !config.GetEnabled() {
		log.Info("SolarWinds Observability APM agent is disabled, skipping startup.")
		return func() {}, nil
	}

	resrc, err := sworesource.CreateResource(resourceAttrs...)
	if err != nil {
		return func() {}, err
	}

	// Setup oboe setting scheduler.
	o := oboe.NewOboe()
	settingsUpdater, err := oboe.NewSettingsUpdater(o)
	if err != nil {
		return func() {}, fmt.Errorf("failed to create settings updater: %w", err)
	}

	ctx := context.Background()
	stopSettingsUpdater := settingsUpdater.Start(ctx)

	smplr, err := sampler.NewSampler(o)
	if err != nil {
		return func() {}, fmt.Errorf("failed to create sampler: %w", err)
	}

	// Parse gRPC endpoint: strip the URL scheme so gRPC gets host:port with default TLS,
	// matching how otelconf handles OTLPGrpc endpoint fields internally.
	grpcEndpoint := config.GetOtelCollector()
	if u, parseErr := url.ParseRequestURI(grpcEndpoint); parseErr == nil && u.Host != "" {
		grpcEndpoint = u.Host
	}

	var grpcHeaders map[string]string
	if token := strings.TrimSpace(config.GetApiToken()); token != "" {
		grpcHeaders = map[string]string{"Authorization": fmt.Sprintf("Bearer %s", token)}
	}

	sdkOpts, err := buildSDKOptions(ctx, otelCfg, grpcEndpoint, grpcHeaders)
	if err != nil {
		stopSettingsUpdater()
		return func() {}, err
	}

	sdk, err := otelconf.NewSDK(sdkOpts...)
	if err != nil {
		stopSettingsUpdater()
		return func() {}, fmt.Errorf("failed to create otelconf SDK: %w", err)
	}

	tracerProvider := sdk.TracerProvider()
	// otelconf's YAML-derived options set a default ParentBased{AlwaysOn} sampler and
	// their own resource, both of which override what we pass via WithTracerProviderOptions.
	// Force-apply the SWO sampler and merge the SWO resource after SDK creation.
	// These reflect-based mutations are safe here because they happen before
	// otel.SetTracerProvider publishes the provider to other goroutines.
	if otelCfg.TracerProvider != nil {
		if tp, ok := tracerProvider.(*sdktrace.TracerProvider); ok {
			setSamplerOnProvider(tp, smplr)
			mergeResourceOnProvider(tp, resrc)
		}
	}
	if otelCfg.MeterProvider != nil {
		if mp, ok := sdk.MeterProvider().(*sdkmetric.MeterProvider); ok {
			mergeResourceOnMeterProvider(mp, resrc)
		}
	}
	if otelCfg.LoggerProvider != nil {
		if lp, ok := sdk.LoggerProvider().(*sdklog.LoggerProvider); ok {
			mergeResourceOnLoggerProvider(lp, resrc)
		}
	}

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(sdk.MeterProvider())
	otelglobal.SetLoggerProvider(sdk.LoggerProvider())
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			sdk.Propagator(),
			&propagator.SolarwindsPropagator{},
		),
	)

	// print all registered providers for visibility
	log.Debugf("TracerProvider: %T", otel.GetTracerProvider())
	log.Debugf("MeterProvider: %T", otel.GetMeterProvider())
	log.Debugf("LoggerProvider: %T", otelglobal.GetLoggerProvider())

	if err = o.RegisterOtelSampleRateMetrics(sdk.MeterProvider()); err != nil {
		stopSettingsUpdater()
		if shutdownErr := sdk.Shutdown(ctx); shutdownErr != nil {
			log.Warningf("failed to shutdown SDK during error cleanup: %v", shutdownErr)
		}
		return func() {}, fmt.Errorf("failed to register sample rate metrics: %w", err)
	}

	if config.GetRuntimeMetrics() {
		if err = runtime.Start(runtime.WithMeterProvider(sdk.MeterProvider())); err != nil {
			stopSettingsUpdater()
			if shutdownErr := sdk.Shutdown(ctx); shutdownErr != nil {
				log.Warningf("failed to shutdown SDK during error cleanup: %v", shutdownErr)
			}
			return func() {}, fmt.Errorf("failed to start runtime metrics: %w", err)
		}
	}

	// Inbound metrics span processor: only registered when the user declared a tracer provider,
	// so there is a real SDK TracerProvider to attach to.
	if otelCfg.TracerProvider != nil {
		registry, registryErr := metrics.NewOtelRegistry(sdk.MeterProvider())
		if registryErr != nil {
			log.Debugf("failed to create metrics registry: %v", registryErr)
			stopSettingsUpdater()
			if shutdownErr := sdk.Shutdown(ctx); shutdownErr != nil {
				log.Warningf("failed to shutdown SDK during error cleanup: %v", shutdownErr)
			}
			return func() {}, fmt.Errorf("failed to create metrics registry: %w", registryErr)
		}
		if tp, ok := tracerProvider.(*sdktrace.TracerProvider); ok {
			tp.RegisterSpanProcessor(processor.NewInboundMetricsSpanProcessor(registry))
		} else {
			log.Warningf("Declarative configuration returned a non-SDK tracer provider (%T), inbound metrics span processor was not registered", tracerProvider)
		}
	}

	return func() {
		stopSettingsUpdater()
		if shutdownErr := sdk.Shutdown(ctx); shutdownErr != nil {
			log.Warningf("failed to shutdown SDK: %v", shutdownErr)
		}
	}, nil
}
