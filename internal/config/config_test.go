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
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/utils"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const TestServiceKey = "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go"

func init() {
	os.Setenv("SW_APM_SERVICE_KEY", TestServiceKey)
	Load()
}

func TestLoadConfig(t *testing.T) {
	key1 := "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:Go"
	key2 := "bbbb315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:Go"

	os.Setenv(envSolarWindsAPMCollector, "example.com:12345")
	os.Setenv(envSolarWindsAPMPrependDomain, "true")
	os.Setenv(envSolarWindsAPMHistogramPrecision, "2")
	os.Setenv(envSolarWindsAPMServiceKey, key1)
	os.Setenv(envSolarWindsAPMEnabled, "true")

	c := NewConfig()
	assert.Equal(t, "example.com:12345", c.GetCollector())
	assert.Equal(t, true, c.PrependDomain)
	assert.Equal(t, 2, c.Precision)
	assert.Equal(t, true, c.Enabled)

	os.Setenv(envSolarWindsAPMCollector, "test.abc:8080")
	os.Setenv(envSolarWindsAPMEnabled, "true")
	os.Setenv(envSolarWindsAPMTracingMode, "always")

	c.Load()
	assert.Equal(t, "test.abc:8080", c.GetCollector())
	assert.Equal(t, true, c.Enabled)
	assert.Equal(t, "enabled", string(c.GetTracingMode()))

	c = NewConfig(
		WithCollector("hello.world"),
		WithServiceKey(key2))
	assert.Equal(t, "hello.world:443", c.GetCollector())
	assert.Equal(t, ToServiceKey(key2), c.GetServiceKey())

	os.Setenv(envSolarWindsAPMServiceKey, key1)
	os.Setenv(envSolarWindsAPMHostnameAlias, "test")
	os.Setenv(envSolarWindsAPMTrustedPath, "test.crt")
	os.Setenv(envSolarWindsAPMEnabled, "invalidValue")
	os.Setenv(envSolarWindsAPMServerlessServiceName, "AWSLambda")
	os.Setenv(envSolarWindsAPMTokenBucketCap, "2.0")
	os.Setenv(envSolarWindsAPMTokenBucketRate, "1.0")
	os.Setenv(envSolarWindsAPMTransactionName, "my-transaction-name")

	c.Load()
	assert.Equal(t, 2.0, c.GetTokenBucketCap())
	assert.Equal(t, 1.0, c.GetTokenBucketRate())
	assert.Equal(t, ToServiceKey(key1), c.GetServiceKey())
	assert.Equal(t, "test", c.GetHostAlias())
	assert.Equal(t, "test.crt", filepath.Base(c.GetTrustedPath()))
	assert.Equal(t, true, c.GetEnabled())
	assert.Equal(t, "", c.GetTransactionName()) // ignore it in non-lambda mode
}

func TestConfig_HasLocalSamplingConfig(t *testing.T) {
	// Set tracing mode
	require.NoError(t, os.Setenv(envSolarWindsAPMTracingMode, "disabled"))
	Load()
	assert.True(t, SamplingConfigured())
	assert.Equal(t, "disabled", string(GetTracingMode()))
	assert.Equal(t, ToInteger(getFieldDefaultValue(&SamplingConfig{}, "SampleRate")), GetSampleRate())

	// No local sampling config
	require.NoError(t, os.Unsetenv(envSolarWindsAPMTracingMode))
	Load()
	assert.False(t, SamplingConfigured())
	assert.Equal(t, getFieldDefaultValue(&SamplingConfig{}, "TracingMode"), string(GetTracingMode()))
	assert.Equal(t, ToInteger(getFieldDefaultValue(&SamplingConfig{}, "SampleRate")), GetSampleRate())

	// Set sample rate to 10000
	require.NoError(t, os.Setenv(envSolarWindsAPMSampleRate, "10000"))
	Load()
	assert.True(t, SamplingConfigured())
	assert.Equal(t, getFieldDefaultValue(&SamplingConfig{}, "TracingMode"), string(GetTracingMode()))
	assert.Equal(t, 10000, GetSampleRate())
}

func TestPrintDelta(t *testing.T) {
	changed := newConfig().reset()
	changed.Collector = "test.com"
	changed.PrependDomain = true
	changed.ReporterProperties.EventFlushInterval = 100

	assert.Equal(t,
		` - Collector (SW_APM_COLLECTOR) = test.com (default: apm.collector.na-01.cloud.solarwinds.com:443)
 - PrependDomain (SW_APM_PREPEND_DOMAIN) = true (default: false)
 - ReporterProperties.EventFlushInterval (SW_APM_EVENTS_FLUSH_INTERVAL) = 100 (default: 2)`,
		getDelta(newConfig().reset(), changed, "").sanitize().String())
}

func TestConfigInit(t *testing.T) {
	c := newConfig()

	// Set them to true, the call to `reset` in next step should reset them to false
	c.Sampling.sampleRateConfigured = true
	c.Sampling.tracingModeConfigured = true

	c.reset()

	defaultC := Config{
		Collector:   defaultSSLCollector,
		ServiceKey:  "",
		TrustedPath: "",
		Sampling: &SamplingConfig{
			TracingMode:           "enabled",
			tracingModeConfigured: false,
			SampleRate:            1000000,
			sampleRateConfigured:  false,
		},
		PrependDomain: false,
		HostAlias:     "",
		Precision:     2,
		ReporterProperties: &ReporterOptions{
			EventFlushInterval:      2,
			MaxReqBytes:             2000 * 1024,
			MetricFlushInterval:     30,
			GetSettingsInterval:     30,
			SettingsTimeoutInterval: 10,
			PingInterval:            20,
			RetryDelayInitial:       500,
			RetryDelayMax:           60,
			RedirectMax:             20,
			RetryLogThreshold:       10,
			MaxRetries:              20,
		},
		SQLSanitize:        0,
		Enabled:            true,
		Ec2MetadataTimeout: 1000,
		DebugLevel:         "warn",
		TriggerTrace:       true,
		Proxy:              "",
		ProxyCertPath:      "",
		RuntimeMetrics:     true,
		TokenBucketCap:     8,
		TokenBucketRate:    0.17,
		ReportQueryString:  true,
	}
	assert.Equal(t, c, &defaultC)
}

func ClearEnvs() {
	for _, kv := range os.Environ() {
		kvSlice := strings.Split(kv, "=")
		k := kvSlice[0]
		os.Unsetenv(k)
	}
}

func SetEnvs(kvs []string) {
	for _, kv := range kvs {
		kvSlice := strings.Split(kv, "=")
		k, v := kvSlice[0], kvSlice[1]
		os.Setenv(k, v)
	}
}

func TestTokenBucketConfigOverRange(t *testing.T) {
	ClearEnvs()

	envs := []string{
		"SW_APM_SERVICE_KEY=ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		"SW_APM_TOKEN_BUCKET_CAPACITY=10",
		"SW_APM_TOKEN_BUCKET_RATE=10",
	}
	SetEnvs(envs)

	c := NewConfig()

	assert.Equal(t, c.TokenBucketCap, 8.0)
	assert.Equal(t, c.TokenBucketRate, 4.0)
}

func TestTokenBucketConfigInvalidValue(t *testing.T) {
	ClearEnvs()

	envs := []string{
		"SW_APM_SERVICE_KEY=ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		"SW_APM_TOKEN_BUCKET_CAPACITY=hello",
		"SW_APM_TOKEN_BUCKET_RATE=hi",
	}
	SetEnvs(envs)

	c := NewConfig()

	assert.Equal(t, c.TokenBucketCap, 8.0)
	assert.Equal(t, c.TokenBucketRate, 0.17)
}

func TestEnvsLoading(t *testing.T) {
	ClearEnvs()

	envs := []string{
		"SW_APM_COLLECTOR=collector.test.com",
		"SW_APM_SERVICE_KEY=ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		"SW_APM_TRUSTEDPATH=/collector.crt",
		"SW_APM_REPORTER=ssl",
		"SW_APM_TRACING_MODE=never",
		"SW_APM_SAMPLE_RATE=1000",
		"SW_APM_PREPEND_DOMAIN=true",
		"SW_APM_HOSTNAME_ALIAS=alias",

		"SW_APM_HISTOGRAM_PRECISION=4",
		"SW_APM_EVENTS_FLUSH_INTERVAL=4",
		"SW_APM_MAX_REQUEST_BYTES=4096000",
		"SW_APM_ENABLED=true",
		"SW_APM_SQL_SANITIZE=0",
		"SW_APM_EC2_METADATA_TIMEOUT=2000",
		"SW_APM_TRIGGER_TRACE=false",
		"SW_APM_PROXY=http://usr/pwd@internal.proxy:3306",
		"SW_APM_PROXY_CERT_PATH=./proxy.pem",
		"SW_APM_RUNTIME_METRICS=true",
		"SW_APM_SERVICE_NAME=LambdaTest",
		"SW_APM_TOKEN_BUCKET_CAPACITY=8",
		"SW_APM_TOKEN_BUCKET_RATE=4",
		"SW_APM_TRANSACTION_NAME=my-transaction-name",
		"SW_APM_REPORT_QUERY_STRING=false",
	}
	SetEnvs(envs)

	envConfig := Config{
		Collector:   "collector.test.com",
		ServiceKey:  "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		TrustedPath: "/collector.crt",
		Sampling: &SamplingConfig{
			TracingMode:           "disabled",
			tracingModeConfigured: true,
			SampleRate:            1000,
			sampleRateConfigured:  true,
		},
		PrependDomain: true,
		HostAlias:     "alias",
		Precision:     2 * 2,
		ReporterProperties: &ReporterOptions{
			EventFlushInterval:      2 * 2,
			MaxReqBytes:             4000 * 1024,
			MetricFlushInterval:     30,
			GetSettingsInterval:     30,
			SettingsTimeoutInterval: 10,
			PingInterval:            20,
			RetryDelayInitial:       500,
			RetryDelayMax:           60,
			RedirectMax:             20,
			RetryLogThreshold:       10,
			MaxRetries:              20,
		},
		SQLSanitize:        0,
		Enabled:            true,
		Ec2MetadataTimeout: 2000,
		DebugLevel:         "warn",
		TriggerTrace:       false,
		Proxy:              "http://usr/pwd@internal.proxy:3306",
		ProxyCertPath:      "./proxy.pem",
		RuntimeMetrics:     true,
		TokenBucketCap:     8,
		TokenBucketRate:    4,
		TransactionName:    "",
		ReportQueryString:  false,
	}

	c := NewConfig()

	assert.Equal(t, c, &envConfig)
}

func TestYamlConfig(t *testing.T) {
	yamlConfig := Config{
		Collector:   "yaml.test.com",
		ServiceKey:  "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189218:go",
		TrustedPath: "/yaml-collector.crt",
		Sampling: &SamplingConfig{
			TracingMode:           "disabled",
			tracingModeConfigured: true,
			SampleRate:            100,
			sampleRateConfigured:  true,
		},
		PrependDomain: true,
		HostAlias:     "yaml-alias",
		Precision:     2 * 3,
		ReporterProperties: &ReporterOptions{
			EventFlushInterval:      2 * 3,
			MaxReqBytes:             2000 * 3 * 1024,
			MetricFlushInterval:     30,
			GetSettingsInterval:     30,
			SettingsTimeoutInterval: 10,
			PingInterval:            20,
			RetryDelayInitial:       500,
			RetryDelayMax:           60,
			RedirectMax:             20,
			RetryLogThreshold:       10,
			MaxRetries:              20,
		},
		TransactionSettings: []TransactionFilter{
			{"url", `\s+\d+\s+`, nil, "disabled"},
			{"url", "", []string{".jpg"}, "disabled"},
		},
		SQLSanitize:        2,
		Enabled:            true,
		Ec2MetadataTimeout: 1500,
		DebugLevel:         "info",
		TriggerTrace:       false,
		Proxy:              "http://usr:pwd@internal.proxy:3306",
		ProxyCertPath:      "./proxy.pem",
		RuntimeMetrics:     true,
		TokenBucketCap:     1.1,
		TokenBucketRate:    2.2,
		TransactionName:    "",
		ReportQueryString:  true,
	}

	out, err := yaml.Marshal(&yamlConfig)
	require.NoError(t, err)

	f, err := os.CreateTemp("", "*-test-config.yaml")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		os.Remove(f.Name())
	}()
	err = os.WriteFile(f.Name(), out, 0644)
	require.NoError(t, err)

	// Test with config file
	ClearEnvs()
	os.Setenv(envSolarWindsAPMConfigFile, f.Name())

	c := NewConfig()
	assert.Equal(t, &yamlConfig, c)

	// Test with both config file and env variables
	envs := []string{
		"SW_APM_COLLECTOR=collector.test.com",
		"SW_APM_SERVICE_KEY=ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		"SW_APM_TRUSTEDPATH=/collector.crt",
		"SW_APM_REPORTER=ssl",
		"SW_APM_TRACING_MODE=never",
		"SW_APM_SAMPLE_RATE=1000",
		"SW_APM_PREPEND_DOMAIN=true",
		"SW_APM_HOSTNAME_ALIAS=alias",
		"SW_APM_HISTOGRAM_PRECISION=4",
		"SW_APM_EVENTS_FLUSH_INTERVAL=4",
		"SW_APM_MAX_REQUEST_BYTES=4096000",
		"SW_APM_ENABLED=true",
		"SW_APM_SQL_SANITIZE=3",
		"SW_APM_SERVICE_NAME=LambdaEnv",
		"SW_APM_TOKEN_BUCKET_CAPACITY=8",
		"SW_APM_TOKEN_BUCKET_RATE=4",
		"SW_APM_TRANSACTION_NAME=transaction-name-from-env",
		"SW_APM_REPORT_QUERY_STRING=false",
	}
	ClearEnvs()
	SetEnvs(envs)
	os.Setenv("SW_APM_CONFIG_FILE", f.Name())

	envConfig := Config{
		Collector:   "collector.test.com",
		ServiceKey:  "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		TrustedPath: "/collector.crt",
		Sampling: &SamplingConfig{
			TracingMode:           "disabled",
			tracingModeConfigured: true,
			SampleRate:            1000,
			sampleRateConfigured:  true,
		},
		PrependDomain: true,
		HostAlias:     "alias",
		Precision:     2 * 2,
		ReporterProperties: &ReporterOptions{
			EventFlushInterval:      2 * 2,
			MaxReqBytes:             4000 * 1024,
			MetricFlushInterval:     30,
			GetSettingsInterval:     30,
			SettingsTimeoutInterval: 10,
			PingInterval:            20,
			RetryDelayInitial:       500,
			RetryDelayMax:           60,
			RedirectMax:             20,
			RetryLogThreshold:       10,
			MaxRetries:              20,
		},
		TransactionSettings: []TransactionFilter{
			{"url", `\s+\d+\s+`, nil, "disabled"},
			{"url", "", []string{".jpg"}, "disabled"},
		},
		SQLSanitize:        3,
		Enabled:            true,
		Ec2MetadataTimeout: 1500,
		DebugLevel:         "info",
		TriggerTrace:       false,
		Proxy:              "http://usr:pwd@internal.proxy:3306",
		ProxyCertPath:      "./proxy.pem",
		RuntimeMetrics:     true,
		TokenBucketCap:     8,
		TokenBucketRate:    4,
		TransactionName:    "",
		ReportQueryString:  false,
	}

	c = NewConfig()
	assert.Equal(t, &envConfig, c)

	os.Unsetenv("SW_APM_CONFIG_FILE")
}

func TestSamplingConfigValidate(t *testing.T) {
	s := &SamplingConfig{
		TracingMode:           "invalid",
		tracingModeConfigured: true,
		SampleRate:            10000000,
		sampleRateConfigured:  true,
	}
	s.validate()
	assert.Equal(t, EnabledTracingMode, s.TracingMode)
	assert.Equal(t, false, s.tracingModeConfigured)
	assert.Equal(t, 1000000, s.SampleRate)
	assert.Equal(t, false, s.sampleRateConfigured)
}

func TestInvalidConfigFile(t *testing.T) {
	var buf utils.SafeBuffer
	var writers []io.Writer

	writers = append(writers, &buf)
	writers = append(writers, os.Stderr)

	log.SetOutput(io.MultiWriter(writers...))
	oldLevel := log.Level()
	log.SetLevel(log.INFO)

	defer func() {
		log.SetOutput(os.Stderr)
		log.SetLevel(oldLevel)
	}()
	f, err := os.CreateTemp("", "*-test-config.json")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		os.Remove(f.Name())
	}()

	ClearEnvs()
	os.Setenv("SW_APM_SERVICE_KEY", "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go")
	os.Setenv("SW_APM_CONFIG_FILE", f.Name())
	require.NoError(t, os.WriteFile(f.Name(), []byte("hello"), 0644))

	_ = NewConfig()
	assert.Contains(t, buf.String(), ErrUnsupportedFormat.Error())
	// OK to ignore this error because we're just making sure this file doesn't exist for the logic below
	_ = os.Remove("/tmp/file-not-exist.yaml")

	buf.Reset()
	ClearEnvs()
	os.Setenv("SW_APM_SERVICE_KEY", "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go")
	os.Setenv("SW_APM_CONFIG_FILE", "/tmp/file-not-exist.yaml")
	_ = NewConfig()
	var exp string
	if runtime.GOOS == "windows" {
		exp = "The system cannot find the path specified."
	} else {
		exp = "no such file or directory"
	}
	assert.Contains(t, buf.String(), exp)
}

func TestInvalidConfig(t *testing.T) {
	var buf utils.SafeBuffer
	var writers []io.Writer

	writers = append(writers, &buf)
	writers = append(writers, os.Stderr)

	log.SetOutput(io.MultiWriter(writers...))
	log.SetLevel(log.INFO)

	defer func() {
		log.SetOutput(os.Stderr)
	}()

	invalid := Config{
		Collector:   "",
		ServiceKey:  "ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		TrustedPath: "",
		Sampling: &SamplingConfig{
			TracingMode:           "disabled",
			tracingModeConfigured: true,
			SampleRate:            1000,
			sampleRateConfigured:  true,
		},
		PrependDomain: true,
		HostAlias:     "alias",
		Precision:     2 * 2,
		ReporterProperties: &ReporterOptions{
			EventFlushInterval:      2 * 2,
			MaxReqBytes:             4000 * 1024,
			MetricFlushInterval:     30,
			GetSettingsInterval:     30,
			SettingsTimeoutInterval: 10,
			PingInterval:            20,
			RetryDelayInitial:       500,
			RetryDelayMax:           60,
			RedirectMax:             20,
			RetryLogThreshold:       10,
			MaxRetries:              20,
		},
		Enabled:            false,
		Ec2MetadataTimeout: 5000,
		DebugLevel:         "info",
	}

	assert.Nil(t, invalid.validate())

	assert.Equal(t, defaultSSLCollector, invalid.Collector)
	assert.Contains(t, buf.String(), "invalid env, discarded - Collector:", buf.String())

	assert.Equal(t, 1000, invalid.Ec2MetadataTimeout)
	assert.Contains(t, buf.String(), "invalid env, discarded - Ec2MetadataTimeout:", buf.String())

	assert.Equal(t, "alias", invalid.HostAlias)
}

// TestConfigDefaultValues is to verify the default values defined in struct Config
// are all correct
func TestConfigDefaultValues(t *testing.T) {
	// A Config object initialized with default values
	c := newConfig().reset()

	// check default log level
	level, ok := log.ToLogLevel(c.DebugLevel)
	assert.Equal(t, level, log.DefaultLevel)
	assert.True(t, ok)

	// check default ssl collector url
	assert.Equal(t, defaultSSLCollector, c.Collector)

	// check the default sample rate
	assert.Equal(t, MaxSampleRate, c.Sampling.SampleRate)
}

func TestTransactionFilter_UnmarshalYAML(t *testing.T) {
	var testCases = []struct {
		filter TransactionFilter
		err    error
	}{
		{TransactionFilter{"invalid", `\s+\d+\s+`, nil, "disabled"}, ErrTFInvalidType},
		{TransactionFilter{"url", `\s+\d+\s+`, nil, "enabled"}, nil},
		{TransactionFilter{"url", `\s+\d+\s+`, nil, "disabled"}, nil},
		{TransactionFilter{"url", "", []string{".jpg"}, "disabled"}, nil},
		{TransactionFilter{"url", `\s+\d+\s+`, []string{".jpg"}, "disabled"}, ErrTFInvalidRegExExt},
		{TransactionFilter{"url", `\s+\d+\s+`, nil, "disabled"}, nil},
		{TransactionFilter{"url", `\s+\d+\s+`, nil, "invalid"}, ErrTFInvalidTracing},
	}

	for idx, testCase := range testCases {
		bytes, err := yaml.Marshal(testCase.filter)
		assert.Nil(t, err, fmt.Sprintf("Case #%d", idx))

		var filter TransactionFilter
		err = yaml.Unmarshal(bytes, &filter)
		assert.Equal(t, testCase.err, err, fmt.Sprintf("Case #%d", idx))
		if err == nil {
			assert.Equal(t, testCase.filter, filter, fmt.Sprintf("Case #%d", idx))
		}
	}
}

func TestTransactionName(t *testing.T) {
	ClearEnvs()

	envs := []string{
		"SW_APM_SERVICE_KEY=ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		"SW_APM_TRANSACTION_NAME=test_name",
	}
	SetEnvs(envs)
	c := NewConfig()
	assert.Equal(t, c.TransactionName, "")

	ClearEnvs()

	envs = []string{
		"SW_APM_SERVICE_KEY=ae38315f6116585d64d82ec2455aa3ec61e02fee25d286f74ace9e4fea189217:go",
		"SW_APM_TRANSACTION_NAME=test_name",
		"AWS_LAMBDA_FUNCTION_NAME=my_function",
		"LAMBDA_TASK_ROOT=some_path",
	}
	SetEnvs(envs)
	c = NewConfig()
	assert.Equal(t, c.TransactionName, "test_name")
}
