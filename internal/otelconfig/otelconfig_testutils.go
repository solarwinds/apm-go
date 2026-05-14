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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	otelglobal "go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
)

// setupLocalListener creates a bare gRPC server on a random local port so
// the OTLP exporter can connect without hanging (~10s timeout on unreachable
// address). The local listener accepts the TCP connection immediately; actual
// RPCs return "unimplemented", which is fine for exporter setup tests.
func setupLocalListener(t *testing.T) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })
	return lis
}

// startMockSettingsServer starts an httptest server that returns a minimal valid
// settings payload so the settings updater does not block startup.
func startMockSettingsServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := map[string]interface{}{
			"flags":     "SAMPLE_START,SAMPLE_THROUGH_ALWAYS,TRIGGER_TRACE",
			"value":     1000000,
			"ttl":       120,
			"timestamp": time.Now().Unix(),
			"arguments": map[string]interface{}{
				"BucketCapacity":               1000000.0,
				"BucketRate":                   1000000.0,
				"MetricsFlushInterval":         30,
				"TriggerRelaxedBucketCapacity": 100.0,
				"TriggerRelaxedBucketRate":     100.0,
				"TriggerStrictBucketCapacity":  10.0,
				"TriggerStrictBucketRate":      10.0,
			},
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// --- reflection helpers for inspecting SDK internals ---

func makeAccessible(v reflect.Value) reflect.Value {
	if !v.IsValid() || v.CanInterface() {
		return v
	}
	if !v.CanAddr() {
		return v
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func reflectValueToString(v reflect.Value) string {
	v = makeAccessible(v)
	if !v.IsValid() {
		return ""
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Map:
		if v.Len() == 0 {
			return ""
		}
		entries := make([]string, 0, v.Len())
		for _, key := range v.MapKeys() {
			entries = append(entries, fmt.Sprintf("%v=%v", key.Interface(), v.MapIndex(key).Interface()))
		}
		sort.Strings(entries)
		return strings.Join(entries, ",")
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return ""
		}
		return reflectValueToString(v.Elem())
	default:
		if v.CanInterface() {
			return fmt.Sprintf("%v", v.Interface())
		}
	}
	return ""
}

func firstNamedFieldValue(v reflect.Value, fieldNames ...string) string {
	v = makeAccessible(v)
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return ""
	}
	for _, fieldName := range fieldNames {
		field := makeAccessible(v.FieldByName(fieldName))
		if !field.IsValid() {
			continue
		}
		if value := reflectValueToString(field); value != "" {
			return value
		}
	}
	return ""
}

type traceExporterDetail struct {
	ProcessorType string
	ExporterType  string
	ClientType    string
	Endpoint      string
	URL           string
	Insecure      string
	Headers       string
}

type meterReaderDetail struct {
	ReaderType   string
	ExporterType string
	Endpoint     string
	URL          string
	Headers      string
}

// tracerProviderExporterDeepDetails reflects into a *sdktrace.TracerProvider to
// extract span processor, exporter, and client details for each registered processor.
func tracerProviderExporterDeepDetails(tp *sdktrace.TracerProvider) []traceExporterDetail {
	if tp == nil {
		return nil
	}

	providerVal := reflect.ValueOf(tp).Elem()
	spanProcessorsField := makeAccessible(providerVal.FieldByName("spanProcessors"))
	if !spanProcessorsField.IsValid() || spanProcessorsField.Kind() != reflect.Struct {
		return nil
	}

	// Call atomic.Pointer.Load() via reflection instead of reading the internal "v"
	// unsafe.Pointer field directly. The direct approach triggers checkptr under -race
	// because it converts a uintptr back to unsafe.Pointer (forbidden pointer arithmetic).
	loadMethod := spanProcessorsField.Addr().MethodByName("Load")
	if !loadMethod.IsValid() {
		return nil
	}
	results := loadMethod.Call(nil)
	if len(results) == 0 || results[0].IsNil() {
		return nil
	}
	statesSlice := makeAccessible(results[0].Elem())
	if statesSlice.Kind() != reflect.Slice {
		return nil
	}

	details := make([]traceExporterDetail, 0, statesSlice.Len())
	for i := 0; i < statesSlice.Len(); i++ {
		stateVal := makeAccessible(statesSlice.Index(i))
		if stateVal.Kind() == reflect.Pointer {
			if stateVal.IsNil() {
				continue
			}
			stateVal = makeAccessible(stateVal.Elem())
		}

		spanProcessorField := makeAccessible(stateVal.FieldByName("sp"))
		if !spanProcessorField.IsValid() || spanProcessorField.Kind() != reflect.Interface || spanProcessorField.IsNil() {
			continue
		}

		spanProcessorVal := spanProcessorField.Elem()
		detail := traceExporterDetail{ProcessorType: spanProcessorVal.Type().String()}

		if spanProcessorVal.Kind() != reflect.Pointer || spanProcessorVal.IsNil() {
			details = append(details, detail)
			continue
		}

		spanProcessorStruct := makeAccessible(spanProcessorVal.Elem())
		exporterField := makeAccessible(spanProcessorStruct.FieldByName("e"))
		if !exporterField.IsValid() || exporterField.Kind() != reflect.Interface || exporterField.IsNil() {
			details = append(details, detail)
			continue
		}

		exporterVal := exporterField.Elem()
		detail.ExporterType = exporterVal.Type().String()

		if exporterVal.Kind() != reflect.Pointer || exporterVal.IsNil() {
			details = append(details, detail)
			continue
		}

		exporterStruct := makeAccessible(exporterVal.Elem())
		detail.Endpoint = firstNamedFieldValue(exporterStruct, "endpoint", "Endpoint", "target", "Target")
		detail.URL = firstNamedFieldValue(exporterStruct, "url", "URL")

		clientField := makeAccessible(exporterStruct.FieldByName("client"))
		if clientField.IsValid() && clientField.Kind() == reflect.Interface && !clientField.IsNil() {
			clientVal := clientField.Elem()
			detail.ClientType = clientVal.Type().String()

			if clientVal.Kind() == reflect.Pointer && !clientVal.IsNil() {
				clientStruct := makeAccessible(clientVal.Elem())
				if detail.Endpoint == "" {
					detail.Endpoint = firstNamedFieldValue(clientStruct, "endpoint", "Endpoint", "target", "Target", "addr", "Addr")
				}
				if detail.URL == "" {
					detail.URL = firstNamedFieldValue(clientStruct, "url", "URL")
				}
				detail.Insecure = firstNamedFieldValue(clientStruct, "insecure", "Insecure")
				detail.Headers = firstNamedFieldValue(clientStruct, "headers", "Headers", "metadata", "Metadata")
			}
		}

		details = append(details, detail)
	}

	return details
}

func findMeterProviderWithPipes(v reflect.Value, depth int) reflect.Value {
	if !v.IsValid() || depth > 8 {
		return reflect.Value{}
	}
	v = makeAccessible(v)
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return reflect.Value{}
		}
		return findMeterProviderWithPipes(v.Elem(), depth+1)
	case reflect.Struct:
		pipesField := makeAccessible(v.FieldByName("pipes"))
		if pipesField.IsValid() && pipesField.Kind() == reflect.Slice {
			return v
		}
		for i := 0; i < v.NumField(); i++ {
			field := makeAccessible(v.Field(i))
			switch field.Kind() {
			case reflect.Interface, reflect.Pointer, reflect.Struct:
				found := findMeterProviderWithPipes(field, depth+1)
				if found.IsValid() {
					return found
				}
			}
		}
	}
	return reflect.Value{}
}

// meterProviderReaderDeepDetails reflects into a MeterProvider to extract reader
// and exporter details for each registered reader pipeline.
func meterProviderReaderDeepDetails(mp any) []meterReaderDetail {
	providerVal := findMeterProviderWithPipes(reflect.ValueOf(mp), 0)
	if !providerVal.IsValid() {
		return nil
	}

	pipesField := makeAccessible(providerVal.FieldByName("pipes"))
	if !pipesField.IsValid() || pipesField.Kind() != reflect.Slice {
		return nil
	}

	details := make([]meterReaderDetail, 0, pipesField.Len())
	for i := 0; i < pipesField.Len(); i++ {
		pipeVal := makeAccessible(pipesField.Index(i))
		if pipeVal.Kind() == reflect.Pointer {
			if pipeVal.IsNil() {
				continue
			}
			pipeVal = makeAccessible(pipeVal.Elem())
		}

		readerField := makeAccessible(pipeVal.FieldByName("reader"))
		if !readerField.IsValid() || readerField.Kind() != reflect.Interface || readerField.IsNil() {
			continue
		}

		readerVal := readerField.Elem()
		detail := meterReaderDetail{ReaderType: readerVal.Type().String()}
		if readerVal.Kind() != reflect.Pointer || readerVal.IsNil() {
			details = append(details, detail)
			continue
		}

		readerStruct := makeAccessible(readerVal.Elem())
		exporterField := makeAccessible(readerStruct.FieldByName("exporter"))
		if !exporterField.IsValid() {
			exporterField = makeAccessible(readerStruct.FieldByName("e"))
		}
		if !exporterField.IsValid() || exporterField.IsNil() {
			details = append(details, detail)
			continue
		}

		exporterVal := exporterField
		if exporterVal.Kind() == reflect.Interface {
			exporterVal = exporterVal.Elem()
		}
		if !exporterVal.IsValid() {
			details = append(details, detail)
			continue
		}

		detail.ExporterType = exporterVal.Type().String()
		if exporterVal.Kind() == reflect.Pointer && !exporterVal.IsNil() {
			exporterStruct := makeAccessible(exporterVal.Elem())
			detail.Endpoint = firstNamedFieldValue(exporterStruct, "endpoint", "Endpoint", "target", "Target", "addr", "Addr")
			detail.URL = firstNamedFieldValue(exporterStruct, "url", "URL")

			clientField := makeAccessible(exporterStruct.FieldByName("client"))
			if clientField.IsValid() && clientField.Kind() == reflect.Interface && !clientField.IsNil() {
				clientVal := clientField.Elem()
				if clientVal.Kind() == reflect.Pointer && !clientVal.IsNil() {
					clientStruct := makeAccessible(clientVal.Elem())
					if detail.Endpoint == "" {
						detail.Endpoint = firstNamedFieldValue(clientStruct, "endpoint", "Endpoint", "target", "Target", "addr", "Addr")
					}
					if detail.URL == "" {
						detail.URL = firstNamedFieldValue(clientStruct, "url", "URL")
					}
					// For gRPC exporters (e.g. otlpmetricgrpc) that store the
					// endpoint only in conn.target on the grpc.ClientConn.
					if detail.Endpoint == "" {
						connField := makeAccessible(clientStruct.FieldByName("conn"))
						if connField.IsValid() && connField.Kind() == reflect.Pointer && !connField.IsNil() {
							connStruct := makeAccessible(connField.Elem())
							detail.Endpoint = firstNamedFieldValue(connStruct, "target", "Target", "addr", "Addr")
						}
					}
					detail.Headers = firstNamedFieldValue(clientStruct, "headers", "Headers", "metadata", "Metadata")
				}
			}
		}

		details = append(details, detail)
	}

	return details
}

// --- shared test constants and helpers ---

const (
	// testServiceKey is a 64-char hex token + service name that satisfies the validator.
	testServiceKey  = "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:otelconf-test"
	testHeaderToken = "Bearer ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217"
)

// startTestAgent writes yamlConfig to a temp file, sets up a mock settings
// server, calls StartWithOtelConf, and registers cleanup via t.Cleanup.
func startTestAgent(t *testing.T, yamlConfig string) {
	t.Helper()

	// Reset OTel globals to noop before each test so that state from a
	// previous test (e.g. a real SDK TracerProvider) does not leak through
	// when this test's StartWithOtelConf returns early (disabled path).
	otel.SetTracerProvider(tracenoop.NewTracerProvider())
	otel.SetMeterProvider(metricnoop.NewMeterProvider())
	otelglobal.SetLoggerProvider(lognoop.NewLoggerProvider())

	srv := startMockSettingsServer(t)
	origSettingsURL := config.SettingsURL
	config.SettingsURL = func() string { return srv.URL }
	t.Cleanup(func() { config.SettingsURL = origSettingsURL })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "otel-config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(yamlConfig), 0600))
	t.Setenv(constants.OtelConfigFileEnv, cfgPath)
	t.Setenv("SW_APM_DISABLED_RESOURCE_DETECTORS", "uams,ec2,azurevm,k8s")

	shutdown, err := StartWithOtelConf()
	require.NoError(t, err)
	t.Cleanup(shutdown)
}

// tracerProviderSamplerDescription reflects into a *sdktrace.TracerProvider and
// returns the registered sampler's Description string.
func tracerProviderSamplerDescription(tp *sdktrace.TracerProvider) string {
	if tp == nil {
		return ""
	}
	v := makeAccessible(reflect.ValueOf(tp).Elem())
	f := makeAccessible(v.FieldByName("sampler"))
	if !f.IsValid() || f.Kind() != reflect.Interface || f.IsNil() {
		return ""
	}
	inner := f.Elem()
	if inner.CanInterface() {
		type describer interface{ Description() string }
		if d, ok := inner.Interface().(describer); ok {
			return d.Description()
		}
	}
	return inner.Type().String()
}

// resourceAttrsFromProvider reads resource attributes directly from the
// TracerProvider via reflection — no span emission required.
func resourceAttrsFromProvider(tp *sdktrace.TracerProvider) map[string]string {
	return resourceFromTracerProvider(tp)
}
