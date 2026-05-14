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
	"reflect"
	"unsafe"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/contrib/otelconf"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// buildSDKOptions constructs the otelconf ConfigurationOptions for each signal
// provider declared in cfg, injecting SWO-managed gRPC exporters when the user
// has not defined their own processors/readers.
// When runtimeMetrics is true and SWO injects the PeriodicReader, a runtime
// producer is attached so that Go runtime metrics are collected on that reader.
func buildSDKOptions(ctx context.Context, cfg *otelconf.OpenTelemetryConfiguration, grpcEndpoint string, grpcHeaders map[string]string, runtimeMetrics bool) ([]otelconf.ConfigurationOption, error) {
	opts := []otelconf.ConfigurationOption{
		otelconf.WithContext(ctx),
	}

	if cfg.TracerProvider != nil && len(cfg.TracerProvider.Processors) == 0 {
		spanExp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(grpcEndpoint),
			otlptracegrpc.WithHeaders(grpcHeaders),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP gRPC span exporter: %w", err)
		}
		opts = append(opts, otelconf.WithTracerProviderOptions(sdktrace.WithBatcher(spanExp)))
	}

	if cfg.MeterProvider != nil && len(cfg.MeterProvider.Readers) == 0 {
		metricExp, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(grpcEndpoint),
			otlpmetricgrpc.WithHeaders(grpcHeaders),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP gRPC metric exporter: %w", err)
		}
		readerOpts := []sdkmetric.PeriodicReaderOption{}
		if runtimeMetrics {
			readerOpts = append(readerOpts, sdkmetric.WithProducer(runtime.NewProducer()))
		}
		opts = append(opts, otelconf.WithMeterProviderOptions(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, readerOpts...))))
	}

	if cfg.LoggerProvider != nil && len(cfg.LoggerProvider.Processors) == 0 {
		logExp, err := otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(grpcEndpoint),
			otlploggrpc.WithHeaders(grpcHeaders),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP gRPC log exporter: %w", err)
		}
		opts = append(opts, otelconf.WithLoggerProviderOptions(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp))))
	}

	return opts, nil
}

// setSamplerOnProvider force-sets the sampler on a *sdktrace.TracerProvider using
// reflection. This is necessary because otelconf applies a default
// ParentBased{AlwaysOn} sampler when initializing the TracerProvider. Since
// we cannot pass a sampler option through buildSDKOptions that would take
// precedence, we force-set the SWO sampler after SDK creation to override it.
func setSamplerOnProvider(tp *sdktrace.TracerProvider, s sdktrace.Sampler) {
	if tp == nil || s == nil {
		return
	}
	v := reflect.ValueOf(tp).Elem()
	f := v.FieldByName("sampler")
	if !f.IsValid() || !f.CanAddr() {
		return
	}
	ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr()))
	ptr.Elem().Set(reflect.ValueOf(s))
}

// mergeResourceOnProvider merges additional into the resource stored in a
// *sdktrace.TracerProvider using reflection. otelconf re-reads OTEL_CONFIG_FILE
// after our options are applied, which resets the resource. We merge via reflection
// after SDK creation to ensure SWO auto-detected resource attrs are present.
// The merge order Merge(additional, current) ensures user-defined YAML attributes
// (current) take precedence over SWO auto-detected values for conflicting keys.
func mergeResourceOnProvider(tp *sdktrace.TracerProvider, additional *sdkresource.Resource) {
	if tp == nil || additional == nil {
		return
	}
	v := reflect.ValueOf(tp).Elem()
	f := v.FieldByName("resource")
	if !f.IsValid() || !f.CanAddr() {
		return
	}
	ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr()))
	current, _ := ptr.Elem().Interface().(*sdkresource.Resource)
	// Merge(additional, current): current (YAML-derived) wins for conflicting keys,
	// while additional (SWO attrs) fills in keys absent from current.
	merged, err := sdkresource.Merge(additional, current)
	if err != nil {
		fmt.Errorf("failed to merge resources for trace_provider: %w", err)
		return
	}
	ptr.Elem().Set(reflect.ValueOf(merged))
}

// mergeResourceOnLoggerProvider merges additional into the resource stored in a
// *sdklog.LoggerProvider using reflection. otelconf re-reads OTEL_CONFIG_FILE after
// our options are applied, which resets the resource. We merge via reflection
// after SDK creation to ensure SWO auto-detected resource attrs are present.
func mergeResourceOnLoggerProvider(lp *sdklog.LoggerProvider, additional *sdkresource.Resource) {
	if lp == nil || additional == nil {
		return
	}
	v := reflect.ValueOf(lp).Elem()
	f := v.FieldByName("resource")
	if !f.IsValid() || !f.CanAddr() {
		return
	}
	ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr()))
	current, _ := ptr.Elem().Interface().(*sdkresource.Resource)
	// Merge(additional, current): current (YAML-derived) wins for conflicting keys.
	merged, err := sdkresource.Merge(additional, current)
	if err != nil {
		fmt.Errorf("failed to merge resources for logger_provider: %w", err)
		return
	}
	ptr.Elem().Set(reflect.ValueOf(merged))
}

// mergeResourceOnMeterProvider merges additional into the resource stored in each
// pipeline of a *sdkmetric.MeterProvider using reflection. The metric SDK stores
// resource per-pipeline (MeterProvider.pipes is []*pipeline, each with its own
// resource field), so all pipelines must be updated.
func mergeResourceOnMeterProvider(mp *sdkmetric.MeterProvider, additional *sdkresource.Resource) {
	if mp == nil || additional == nil {
		return
	}
	v := reflect.ValueOf(mp).Elem()
	pipesField := v.FieldByName("pipes")
	if !pipesField.IsValid() || !pipesField.CanAddr() {
		return
	}
	// pipelines is []*pipeline; iterate and merge resource into each pipeline.
	pipes := reflect.NewAt(pipesField.Type(), unsafe.Pointer(pipesField.UnsafeAddr())).Elem()
	for i := 0; i < pipes.Len(); i++ {
		pipe := pipes.Index(i)
		if pipe.Kind() != reflect.Ptr || pipe.IsNil() {
			continue
		}
		resField := pipe.Elem().FieldByName("resource")
		if !resField.IsValid() || !resField.CanAddr() {
			continue
		}
		resPtr := reflect.NewAt(resField.Type(), unsafe.Pointer(resField.UnsafeAddr()))
		current, _ := resPtr.Elem().Interface().(*sdkresource.Resource)
		// Merge(additional, current): current (YAML-derived) wins for conflicting keys.
		merged, err := sdkresource.Merge(additional, current)
		if err != nil {
			fmt.Errorf("failed to merge resources for meter_provider: %w", err)
			continue
		}
		resPtr.Elem().Set(reflect.ValueOf(merged))
	}
}

// For testing only: resourceFromTracerProvider reads the resource stored inside a
// *sdktrace.TracerProvider directly via reflection, returning the resource
// attributes as a map. This is the canonical way to verify resource attributes
// in tests without having to emit a span.
func resourceFromTracerProvider(tp *sdktrace.TracerProvider) map[string]string {
	if tp == nil {
		return nil
	}
	v := reflect.ValueOf(tp).Elem()
	f := v.FieldByName("resource")
	if !f.IsValid() || !f.CanAddr() {
		return nil
	}
	ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr()))
	res, _ := ptr.Elem().Interface().(*sdkresource.Resource)
	if res == nil {
		return nil
	}
	attrs := make(map[string]string, res.Len())
	for _, kv := range res.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}
	return attrs
}
