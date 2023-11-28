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
// Package config is responsible for loading the configuration from various
// sources, e.g., environment variables, configuration files and user input.
// It also accepts dynamic settings from the collector server.
//
// In order to add a new configuration item, you need to:
//   - add a field to the Config struct and assign the corresponding env variable
//     name and the default value via struct tags.
//   - add validation code to method `Config.validate()` (optional).
//   - add a method to retrieve the config value and a wrapper for the default
//     global variable `conf` (see wrappers.go).
package config

import (
	"fmt"
	"github.com/solarwinds/apm-go/internal/log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	// MaxSampleRate is the maximum sample rate we can have
	MaxSampleRate = 1000000
	// MinSampleRate is the minimum sample rate we can have
	MinSampleRate = 0
	// max config file size = 1MB
	maxConfigFileSize = 1024 * 1024
	// the default collector url
	defaultSSLCollector    = "apm.collector.na-01.cloud.solarwinds.com:443"
	maxTokenBucketCapacity = 8
	maxTokenBucketRate     = 4
)

// The environment variables
const (
	envSolarWindsAPMCollector             = "SW_APM_COLLECTOR"
	envSolarWindsAPMServiceKey            = "SW_APM_SERVICE_KEY"
	envSolarWindsAPMTrustedPath           = "SW_APM_TRUSTEDPATH"
	envSolarWindsAPMReporter              = "SW_APM_REPORTER"
	envSolarWindsAPMTracingMode           = "SW_APM_TRACING_MODE"
	envSolarWindsAPMSampleRate            = "SW_APM_SAMPLE_RATE"
	envSolarWindsAPMPrependDomain         = "SW_APM_PREPEND_DOMAIN"
	envSolarWindsAPMHostnameAlias         = "SW_APM_HOSTNAME_ALIAS"
	envSolarWindsAPMHistogramPrecision    = "SW_APM_HISTOGRAM_PRECISION"
	envSolarWindsAPMEventsFlushInterval   = "SW_APM_EVENTS_FLUSH_INTERVAL"
	envSolarWindsAPMMaxReqBytes           = "SW_APM_MAX_REQUEST_BYTES"
	envSolarWindsAPMEnabled               = "SW_APM_ENABLED"
	envSolarWindsAPMConfigFile            = "SW_APM_CONFIG_FILE"
	envSolarWindsAPMServerlessServiceName = "SW_APM_SERVICE_NAME"
	envSolarWindsAPMTokenBucketCap        = "SW_APM_TOKEN_BUCKET_CAPACITY"
	envSolarWindsAPMTokenBucketRate       = "SW_APM_TOKEN_BUCKET_RATE"
	envSolarWindsAPMTransactionName       = "SW_APM_TRANSACTION_NAME"
)

// Errors
var (
	ErrUnsupportedFormat = errors.New("unsupported format")
	ErrFileTooLarge      = errors.New("file size exceeds limit")
	ErrInvalidServiceKey = errors.New(fullTextInvalidServiceKey)
)

// Config is the struct to define the agent configuration. The configuration
// options in this struct (excluding those from ReporterOptions) are not
// intended for dynamically updating.
type Config struct {
	sync.RWMutex `yaml:"-"`

	// Collector defines the host and port of the SolarWinds Observability collector
	Collector string `yaml:"Collector,omitempty" env:"SW_APM_COLLECTOR" default:"apm.collector.na-01.cloud.solarwinds.com:443"`

	// ServiceKey defines the service key and service name
	ServiceKey string `yaml:"ServiceKey,omitempty" env:"SW_APM_SERVICE_KEY"`

	// The file path of the cert file for gRPC connection
	TrustedPath string `yaml:"TrustedPath,omitempty" env:"SW_APM_TRUSTEDPATH"`

	// The reporter type, ssl or serverless
	ReporterType string `yaml:"ReporterType,omitempty" env:"SW_APM_REPORTER" default:"ssl"`

	Sampling *SamplingConfig `yaml:"Sampling,omitempty"`

	// Whether the domain should be prepended to the transaction name.
	PrependDomain bool `yaml:"PrependDomain,omitempty" env:"SW_APM_PREPEND_DOMAIN"`

	// The alias of the hostname
	HostAlias string `yaml:"HostAlias,omitempty" env:"SW_APM_HOSTNAME_ALIAS"`

	// The precision of the histogram
	Precision int `yaml:"Precision,omitempty" env:"SW_APM_HISTOGRAM_PRECISION" default:"2"`

	// The SQL sanitization level
	SQLSanitize int `yaml:"SQLSanitize,omitempty" env:"SW_APM_SQL_SANITIZE" default:"0"`

	// The reporter options
	ReporterProperties *ReporterOptions `yaml:"ReporterProperties,omitempty"`

	// The transaction filtering config
	TransactionSettings []TransactionFilter `yaml:"TransactionSettings,omitempty"`

	Enabled bool `yaml:"Enabled,omitempty" env:"SW_APM_ENABLED" default:"true"`

	// EC2 metadata retrieval timeout in milliseconds
	Ec2MetadataTimeout int `yaml:"Ec2MetadataTimeout,omitempty" env:"SW_APM_EC2_METADATA_TIMEOUT" default:"1000"`

	// The default log level. It should follow the level defined in log.DefaultLevel
	DebugLevel string `yaml:"DebugLevel,omitempty" env:"SW_APM_DEBUG_LEVEL" default:"warn"`

	// The flag for trigger trace. It's enabled by default.
	TriggerTrace bool `yaml:"TriggerTrace" env:"SW_APM_TRIGGER_TRACE" default:"true"`

	// Url of the HTTP/HTTPS proxy in the format of "scheme://<username>:<password>@<host>:<port>"
	Proxy string `yaml:"Proxy,omitempty" env:"SW_APM_PROXY"`
	// Cert path for the HTTP/HTTPS proxy
	ProxyCertPath string `yaml:"ProxyCertPath" env:"SW_APM_PROXY_CERT_PATH"`
	// Report runtime metrics or not
	RuntimeMetrics bool `yaml:"RuntimeMetrics" env:"SW_APM_RUNTIME_METRICS" default:"true"`
	// ReportQueryString indicates if the query string should be reported as part of the URL
	ReportQueryString bool    `yaml:"ReportQueryString" env:"SW_APM_REPORT_QUERY_STRING" default:"true"`
	TokenBucketCap    float64 `yaml:"TokenBucketCap" env:"SW_APM_TOKEN_BUCKET_CAPACITY" default:"8"`
	TokenBucketRate   float64 `yaml:"TokenBucketRate" env:"SW_APM_TOKEN_BUCKET_RATE" default:"0.17"`
	// The user-defined transaction name. It's only available in the AWS Lambda environment.
	TransactionName string `yaml:"TransactionName" env:"SW_APM_TRANSACTION_NAME"`
}

// SamplingConfig defines the configuration options for the sampling decision
type SamplingConfig struct {
	// The tracing mode
	TracingMode TracingMode `yaml:"TracingMode,omitempty" env:"SW_APM_TRACING_MODE" default:"enabled"`
	// If the tracing mode is configured explicitly
	tracingModeConfigured bool `yaml:"-"`

	// The sample rate
	SampleRate int `yaml:"SampleRate,omitempty" env:"SW_APM_SAMPLE_RATE" default:"1000000"`
	// If the sample rate is configured explicitly
	sampleRateConfigured bool `yaml:"-"`
}

// FilterType defines the type of the transaction filter
type FilterType string

const (
	// URL based filter
	URL FilterType = "url"
)

// TracingMode defines the tracing mode which is either `enabled` or `disabled`
type TracingMode string

const (
	// EnabledTracingMode means tracing is enabled
	EnabledTracingMode TracingMode = "enabled"
	// DisabledTracingMode means tracing is disabled
	DisabledTracingMode TracingMode = "disabled"

	UnknownTracingMode TracingMode = "unknown"
)

// TransactionFilter defines the transaction filtering based on a filter type.
type TransactionFilter struct {
	Type       FilterType  `yaml:"Type"`
	RegEx      string      `yaml:"RegEx,omitempty"`
	Extensions []string    `yaml:"Extensions,omitempty"`
	Tracing    TracingMode `yaml:"Tracing"`
}

// TransactionFilter unmarshal errors
var (
	ErrTFInvalidType     = errors.New("invalid Type")
	ErrTFInvalidTracing  = errors.New("invalid Tracing")
	ErrTFInvalidRegExExt = errors.New("must set either RegEx or Extensions, but not both")
)

// UnmarshalYAML is the customized unmarshal method for TransactionFilter
func (f *TransactionFilter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	initStruct(f)
	var aux = struct {
		Type       FilterType  `yaml:"Type"`
		RegEx      string      `yaml:"RegEx,omitempty"`
		Extensions []string    `yaml:"Extensions,omitempty"`
		Tracing    TracingMode `yaml:"Tracing"`
	}{}

	if err := unmarshal(&aux); err != nil {
		return errors.Wrap(err, "failed to unmarshal TransactionFilter")
	}
	if aux.Type != URL {
		return ErrTFInvalidType
	}
	if aux.Tracing != EnabledTracingMode && aux.Tracing != DisabledTracingMode {
		return ErrTFInvalidTracing
	}
	if (aux.RegEx == "") == (aux.Extensions == nil) {
		return ErrTFInvalidRegExExt
	}

	f.Type = aux.Type
	f.RegEx = aux.RegEx
	f.Extensions = aux.Extensions
	f.Tracing = aux.Tracing
	return nil
}

// Configured returns if either the tracing mode or the sampling rate has been configured
func (s *SamplingConfig) Configured() bool {
	return s.tracingModeConfigured || s.sampleRateConfigured
}

// UnmarshalYAML is the customized unmarshal method for SamplingConfig
func (s *SamplingConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	initStruct(s)
	var aux = struct {
		TracingMode TracingMode `yaml:"TracingMode"`
		SampleRate  int         `yaml:"SampleRate"`
	}{
		TracingMode: "Invalid",
		SampleRate:  -1,
	}
	if err := unmarshal(&aux); err != nil {
		return errors.Wrap(err, "failed to unmarshal SamplingConfig")
	}

	if aux.TracingMode != "Invalid" {
		s.SetTracingMode(aux.TracingMode)
	}
	if aux.SampleRate != -1 {
		s.SetSampleRate(aux.SampleRate)
	}
	return nil
}

// ResetTracingMode resets the tracing mode to its default value and clear the flag.
func (s *SamplingConfig) ResetTracingMode() {
	s.TracingMode = TracingMode(getFieldDefaultValue(s, "TracingMode"))
	s.tracingModeConfigured = false
}

// ResetSampleRate resets the sample rate to its default value and clear the flag.
func (s *SamplingConfig) ResetSampleRate() {
	s.SampleRate = ToInteger(getFieldDefaultValue(s, "SampleRate"))
	s.sampleRateConfigured = false
}

func (s *SamplingConfig) validate() {
	if ok := IsValidTracingMode(s.TracingMode); !ok {
		log.Info(InvalidEnv("TracingMode", string(s.TracingMode)))
		s.ResetTracingMode()
	}
	if ok := IsValidSampleRate(s.SampleRate); !ok {
		log.Info(InvalidEnv("SampleRate", strconv.Itoa(s.SampleRate)))
		s.ResetSampleRate()
	}
}

// SetTracingMode assigns the tracing mode and set the corresponding flag.
// Note: Do not change the method name as it (`Set`+Field name) is used in method
// `loadEnvsInternal` to assign the values loaded from env variables dynamically.
func (s *SamplingConfig) SetTracingMode(mode TracingMode) {
	s.TracingMode = NormalizeTracingMode(mode)
	s.tracingModeConfigured = true
}

// SetSampleRate assigns the sample rate and set the corresponding flag.
// Note: Do not change the method name as it (`Set`+Field name) is used in method
// `loadEnvsInternal` to assign the values loaded from env variables dynamically.
func (s *SamplingConfig) SetSampleRate(rate int) {
	s.SampleRate = rate
	s.sampleRateConfigured = true
}

// Get the value of the `default` tag of a field in the struct.
func getFieldDefaultValue(i interface{}, fieldName string) string {
	iv := reflect.Indirect(reflect.ValueOf(i))
	if iv.Kind() != reflect.Struct {
		panic("calling getFieldDefaultValue with non-struct type")
	}

	field, ok := iv.Type().FieldByName(fieldName)
	if !ok {
		panic(fmt.Sprintf("invalid field: %s", fieldName))
	}

	return field.Tag.Get("default")
}

// Option is a function type that accepts a Config pointer and
// applies the configuration option it defines.
type Option func(c *Config)

// WithCollector defines a Config option for collector address.
func WithCollector(collector string) Option {
	return func(c *Config) {
		c.Collector = collector
	}
}

// WithServiceKey defines a Config option for the service key.
func WithServiceKey(key string) Option {
	return func(c *Config) {
		c.ServiceKey = key
	}
}

// NewConfig initializes a Config object and override default values with options
// provided as arguments. It may print errors if there are invalid values in the
// configuration file or the environment variables.
//
// If there is a fatal error (e.g., invalid config file), it will return a config
// object with default values.
func NewConfig(opts ...Option) *Config {
	return newConfig().Load(opts...)
}

const (
	fullTextInvalidServiceKey = `
	    **No valid service key (defined as token:service_name) is found.** 
	
	Please check SolarWinds Observability dashboard for your token and use a valid service name.
	A valid service name should be shorter than 256 characters and contain only 
	valid characters: [a-z0-9.:_-]. 

	Also note that the agent may convert the service name by removing invalid 
	characters and replacing spaces with hyphens, so the finalized service key 
	may be different from your setting.`
)

// hasLambdaEnv checks if the AWS Lambda env var is set.
func hasLambdaEnv() bool {
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" && os.Getenv("LAMBDA_TASK_ROOT") != ""
}

func (c *Config) validate() error {
	if ok := IsValidHost(c.Collector); !ok {
		log.Info(InvalidEnv("Collector", c.Collector))
		c.Collector = getFieldDefaultValue(c, "Collector")
	}

	if ok := IsValidFile(c.TrustedPath); !ok {
		log.Info(InvalidEnv("TrustedPath", c.TrustedPath))
		c.TrustedPath = getFieldDefaultValue(c, "TrustedPath")
	}

	if ok := IsValidEc2MetadataTimeout(c.Ec2MetadataTimeout); !ok {
		log.Info(InvalidEnv("Ec2MetadataTimeout", strconv.Itoa(c.Ec2MetadataTimeout)))
		t, _ := strconv.Atoi(getFieldDefaultValue(c, "Ec2MetadataTimeout"))
		c.Ec2MetadataTimeout = t
	}

	if hasLambdaEnv() {
		c.ReporterType = reporterTypeServerless
	} else {
		c.ReporterType = strings.ToLower(strings.TrimSpace(c.ReporterType))
	}
	if ok := IsValidReporterType(c.ReporterType); !ok {
		log.Info(InvalidEnv("ReporterType", c.ReporterType))
		c.ReporterType = getFieldDefaultValue(c, "ReporterType")
	}

	if c.TransactionName != "" && c.ReporterType != reporterTypeServerless {
		log.Info(InvalidEnv("TransactionName", c.TransactionName))
		c.TransactionName = getFieldDefaultValue(c, "TransactionName")
	}

	if c.ReporterType != reporterTypeServerless {
		if c.ServiceKey != "" {
			c.ServiceKey = ToServiceKey(c.ServiceKey)
			if ok := IsValidServiceKey(c.ServiceKey); !ok {
				return errors.Wrap(ErrInvalidServiceKey, fmt.Sprintf("service key: \"%s\"", c.ServiceKey))
			}
		}
	}

	c.Sampling.validate()

	if ok := IsValidHostnameAlias(c.HostAlias); !ok {
		log.Warning(InvalidEnv("HostAlias", c.HostAlias))
		c.HostAlias = getFieldDefaultValue(c, "HostAlias")
	}

	if _, valid := log.ToLogLevel(c.DebugLevel); !valid {
		log.Warning(InvalidEnv("DebugLevel", c.DebugLevel))
		c.DebugLevel = getFieldDefaultValue(c, "DebugLevel")
	}

	if valid := IsValidTokenBucketCap(c.TokenBucketCap); !valid {
		log.Warning(InvalidEnv("TokenBucketCap", fmt.Sprintf("%f", c.TokenBucketCap)))
		if c.TokenBucketCap < 0 {
			c.TokenBucketCap = 0
		} else {
			c.TokenBucketCap = maxTokenBucketCapacity
		}
	}

	if valid := IsValidTokenBucketRate(c.TokenBucketRate); !valid {
		log.Warning(InvalidEnv("TokenBucketRate", fmt.Sprintf("%f", c.TokenBucketRate)))
		if c.TokenBucketRate < 0 {
			c.TokenBucketRate = 0
		} else {
			c.TokenBucketRate = maxTokenBucketRate
		}
	}

	return c.ReporterProperties.validate()
}

// Load reads configuration from config file and environment variables.
func (c *Config) Load(opts ...Option) *Config {
	c.Lock()
	defer c.Unlock()

	c.reset()

	if err := c.loadConfigFile(); err != nil {
		log.Warning(errors.Wrap(err, "config file load error").Error())
		return c.resetThenDisable()
	}
	c.loadEnvs()

	for _, opt := range opts {
		opt(c)
	}

	if !c.Enabled {
		return c.resetThenDisable()
	}

	if err := c.validate(); err != nil {
		log.Warning(errors.Wrap(err, "validation error").Error())
		return c.resetThenDisable()
	}

	return c
}

func (c *Config) resetThenDisable() *Config {
	c.reset()
	c.Enabled = false
	return c
}

func (c *Config) GetDelta() *Delta {
	return getDelta(newConfig().reset(), c, "").sanitize()
}

// DeltaItem defines a delta item  of two Config objects
type DeltaItem struct {
	key        string
	env        string
	value      string
	defaultVal string
}

// Delta defines the overall delta of two Config objects
type Delta struct {
	delta []DeltaItem
}

func (d *Delta) add(item ...DeltaItem) {
	d.delta = append(d.delta, item...)
}

func (d *Delta) items() []DeltaItem {
	return d.delta
}

func (d *Delta) sanitize() *Delta {
	for idx := range d.delta {
		// mask the sensitive service key
		if d.delta[idx].key == "ServiceKey" {
			d.delta[idx].value = MaskServiceKey(d.delta[idx].value)
		}
	}
	return d
}

func (d *Delta) String() string {
	var s []string
	for _, item := range d.delta {
		s = append(s, fmt.Sprintf(" - %s (%s) = %s (default: %s)",
			item.key,
			item.env,
			item.value,
			item.defaultVal))
	}
	return strings.Join(s, "\n")
}

// getDelta compares two instances of the same struct and returns the delta.
func getDelta(base, changed interface{}, keyPrefix string) *Delta {
	delta := &Delta{}

	baseVal := reflect.Indirect(reflect.ValueOf(base))
	changedVal := reflect.Indirect(reflect.ValueOf(changed))

	if baseVal.Kind() != reflect.Struct || changedVal.Kind() != reflect.Struct {
		return delta
	}

	for i := 0; i < changedVal.NumField(); i++ {
		typeFieldChanged := changedVal.Type().Field(i)
		if typeFieldChanged.Anonymous {
			continue
		}

		fieldChanged := reflect.Indirect(changedVal.Field(i))
		fieldBase := reflect.Indirect(baseVal.Field(i))

		if fieldChanged.Kind() == reflect.Struct {
			prefix := typeFieldChanged.Name
			baseField := baseVal.Field(i).Interface()
			changedField := changedVal.Field(i).Interface()

			subDelta := getDelta(baseField, changedField, prefix)
			delta.add(subDelta.items()...)
		} else {
			if !fieldChanged.CanSet() { // only collect the exported fields
				continue
			}

			if !reflect.DeepEqual(fieldBase.Interface(), fieldChanged.Interface()) {
				keyName := typeFieldChanged.Name
				if keyPrefix != "" {
					keyName = fmt.Sprintf("%s.%s", keyPrefix, typeFieldChanged.Name)
				}
				kv := DeltaItem{
					key:        keyName,
					env:        typeFieldChanged.Tag.Get("env"),
					value:      fmt.Sprintf("%+v", fieldChanged.Interface()),
					defaultVal: fmt.Sprintf("%+v", fieldBase.Interface()),
				}
				delta.add(kv)
			}
		}
	}
	return delta
}

func newConfig() *Config {
	return &Config{
		Sampling:           &SamplingConfig{},
		ReporterProperties: &ReporterOptions{},
	}
}

// reset reads the field tag `default` from the struct definition and initialize
// the struct object with the default value.
func (c *Config) reset() *Config {
	return initStruct(c).(*Config)
}

// initStruct initializes the struct with the default values defined in the struct
// tags.
// The input must be a pointer to a settable struct object.
func initStruct(c interface{}) interface{} {
	cVal := reflect.Indirect(reflect.ValueOf(c))
	cType := cVal.Type()
	cPtrType := reflect.ValueOf(c).Type()

	for i := 0; i < cVal.NumField(); i++ {
		fieldVal := reflect.Indirect(cVal.Field(i))
		field := cType.Field(i)

		if field.Anonymous || !fieldVal.CanSet() {
			continue
		}
		if fieldVal.Kind() == reflect.Struct {
			// Need to use its pointer, otherwise it won't be addressable after
			// passed into the nested method
			initStruct(getValPtr(cVal.Field(i)).Interface())
		} else {
			tagVal := getFieldDefaultValue(c, field.Name)
			defaultVal, _ := stringToValue(tagVal, field.Type)

			resetMethod := fmt.Sprintf("%s%s", "Reset", field.Name)
			var prefix string

			// The *T type is required to fetch the method defined with *T
			if _, ok := cPtrType.MethodByName(resetMethod); ok {
				prefix = "Reset"
			} else {
				prefix = "Set"
			}
			setField(c, prefix, field, defaultVal)
		}
	}

	return c
}

// setField assigns `val` to struct c's field specified by the name `field`. It
// first checks if there is a setter method `setterPrefix+FieldName` for this
// field, and calls the setter if so. Otherwise it sets the value directly via the
// reflect.Value.Set method.
//
// The setter, if provided, is preferred as there may be some extra work around,
// for example it may need to acquire a mutex, or set some flags.
// The `val` must have the same dynamic type as the field, otherwise it will panic.
//
// The dynamic type of the first argument `c` must be a pointer to a struct object.
func setField(c interface{}, prefix string, field reflect.StructField, val reflect.Value) {
	cVal := reflect.Indirect(reflect.ValueOf(c))
	if cVal.Kind() != reflect.Struct {
		return
	}

	fieldVal := reflect.Indirect(cVal.FieldByName(field.Name))
	if !fieldVal.IsValid() {
		return
	}

	fieldKind := field.Type.Kind()
	if !fieldVal.CanSet() || field.Anonymous || fieldKind == reflect.Struct {
		log.Warningf("Failed to set field: %s val: %v", field.Name, val.Interface())
		return
	}

	setMethodName := fmt.Sprintf("%s%s", prefix, field.Name)
	setMethodV := reflect.ValueOf(c).MethodByName(setMethodName)
	// setMethod may be invalid, but we call setMethodV.IsValid() first before
	// all the other methods so it should be safe.
	setMethodM, _ := reflect.TypeOf(c).MethodByName(setMethodName)
	setMethodT := setMethodM.Type

	// Call the setter if we have found a valid one
	if setMethodV.IsValid() {
		var in []reflect.Value
		// The method should not have more than 2 parameters, while the receiver
		// is the first parameter.
		if setMethodT.NumIn() == 2 && setMethodT.In(1).Kind() == fieldKind {
			in = append(in, val)
		}
		setMethodV.Call(in)
	} else {
		fieldVal.Set(val)
	}
}

// loadEnvs loads environment variable values and update the Config object.
func (c *Config) loadEnvs() {
	loadEnvsInternal(c)
}

// getValPtr returns the pointer value of the input argument if it's not a Ptr
// The val must be addressable, otherwise it will panic.
func getValPtr(val reflect.Value) reflect.Value {
	if val.Kind() == reflect.Ptr {
		return val
	}
	return val.Addr()
}

// getConfigPath returns the absolute path of the config file.
func (c *Config) getConfigPath() string {
	path, ok := os.LookupEnv(envSolarWindsAPMConfigFile)
	if ok {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		} else {
			log.Warningf("Ignore config file %s: %s", path, err)
		}
	}

	candidates := []string{
		"./solarwinds-apm-goagent.yaml",
		"./solarwinds-apm-goagent.yml",
		"/solarwinds-apm-goagent.yaml",
		"/solarwinds-apm-goagent.yml",
	}

	for _, file := range candidates {
		abs, err := filepath.Abs(file)
		if err != nil {
			continue
		}
		if _, e := os.Stat(abs); e != nil {
			continue
		}
		return abs
	}

	return ""
}

func (c *Config) loadYaml(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "loadYaml")
	}

	// A pointer field may be assigned with nil in unmarshal, so just keep the
	// old default value and re-assign it later.
	origSampling := c.Sampling
	origReporterProperties := c.ReporterProperties

	// The config struct is modified in place so we won't tolerate any error
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("loadYaml: %s", path))
	}

	if c.Sampling == nil {
		c.Sampling = origSampling
	}
	if c.ReporterProperties == nil {
		c.ReporterProperties = origReporterProperties
	}

	return nil
}

func (c *Config) checkFileSize(path string) error {
	file, err := os.Stat(path)
	if err != nil {
		return errors.Wrap(err, "checkFileSize")
	}
	size := file.Size()
	if size > maxConfigFileSize {
		return errors.Wrap(ErrFileTooLarge, fmt.Sprintf("File size: %d", size))
	}
	return nil
}

// loadConfigFile loads configuration from the config file.
func (c *Config) loadConfigFile() error {
	path := c.getConfigPath()
	if path == "" {
		return nil
	}

	if err := c.checkFileSize(path); err != nil {
		return errors.Wrap(err, "loadConfigFile")
	}
	ext := filepath.Ext(path)

	switch ext {
	case ".yml", ".yaml":
		return c.loadYaml(path)
	default:
		return errors.Wrap(ErrUnsupportedFormat, path)
	}
}

// GetCollector returns the collector address
func (c *Config) GetCollector() string {
	c.RLock()
	defer c.RUnlock()
	return c.Collector
}

// GetServiceKey returns the service key
func (c *Config) GetServiceKey() string {
	c.RLock()
	defer c.RUnlock()
	return c.ServiceKey
}

// GetTrustedPath returns the file path of the cert file
func (c *Config) GetTrustedPath() string {
	c.RLock()
	defer c.RUnlock()
	return c.TrustedPath
}

// GetReporterType returns the reporter type
func (c *Config) GetReporterType() string {
	c.RLock()
	defer c.RUnlock()
	return c.ReporterType
}

// GetTracingMode returns the local tracing mode
func (c *Config) GetTracingMode() TracingMode {
	c.RLock()
	defer c.RUnlock()
	return c.Sampling.TracingMode
}

// GetSampleRate returns the local sample rate
func (c *Config) GetSampleRate() int {
	c.RLock()
	defer c.RUnlock()
	return c.Sampling.SampleRate
}

// SamplingConfigured returns if tracing mode or sampling rate is configured
func (c *Config) SamplingConfigured() bool {
	c.RLock()
	defer c.RUnlock()
	return c.Sampling.Configured()
}

// GetPrependDomain returns the prepend domain config
func (c *Config) GetPrependDomain() bool {
	c.RLock()
	defer c.RUnlock()
	return c.PrependDomain
}

// GetHostAlias returns the host alias
func (c *Config) GetHostAlias() string {
	c.RLock()
	defer c.RUnlock()
	return c.HostAlias
}

// GetPrecision returns the histogram precision
func (c *Config) GetPrecision() int {
	c.RLock()
	defer c.RUnlock()
	return c.Precision
}

// GetEnabled returns if the agent is enabled
func (c *Config) GetEnabled() bool {
	c.RLock()
	defer c.RUnlock()
	return c.Enabled
}

// GetReporter returns the reporter options struct
func (c *Config) GetReporter() *ReporterOptions {
	c.RLock()
	defer c.RUnlock()
	return c.ReporterProperties
}

// GetEc2MetadataTimeout returns the EC2 metadata retrieval timeout in milliseconds
func (c *Config) GetEc2MetadataTimeout() int {
	c.RLock()
	defer c.RUnlock()
	return c.Ec2MetadataTimeout
}

// GetDebugLevel returns the global logging level. Note that it may return an
// empty string
func (c *Config) GetDebugLevel() string {
	c.RLock()
	defer c.RUnlock()
	return c.DebugLevel
}

// GetTriggerTrace returns the trigger trace configuration
func (c *Config) GetTriggerTrace() bool {
	c.RLock()
	defer c.RUnlock()
	return c.TriggerTrace
}

// GetProxy returns the HTTP/HTTPS proxy url
func (c *Config) GetProxy() string {
	c.RLock()
	defer c.RUnlock()
	return c.Proxy
}

// GetProxyCertPath returns the proxy's certificate path
func (c *Config) GetProxyCertPath() string {
	c.RLock()
	defer c.RUnlock()
	return c.ProxyCertPath
}

// GetRuntimeMetrics returns the runtime metrics flag
func (c *Config) GetRuntimeMetrics() bool {
	c.RLock()
	defer c.RUnlock()
	return c.RuntimeMetrics
}

// GetTokenBucketCap returns the token bucket capacity
func (c *Config) GetTokenBucketCap() float64 {
	c.RLock()
	defer c.RUnlock()
	return c.TokenBucketCap
}

// GetTokenBucketRate returns the token bucket rate
func (c *Config) GetTokenBucketRate() float64 {
	c.RLock()
	defer c.RUnlock()
	return c.TokenBucketRate
}

// GetReportQueryString returns the ReportQueryString flag
func (c *Config) GetReportQueryString() bool {
	c.RLock()
	defer c.RUnlock()
	return c.ReportQueryString
}

// GetTransactionFiltering returns the transaction filtering config
func (c *Config) GetTransactionFiltering() []TransactionFilter {
	c.RLock()
	defer c.RUnlock()
	return c.TransactionSettings
}

// GetTransactionName returns the user-defined transaction name. It's only available
// in the AWS Lambda environment.
func (c *Config) GetTransactionName() string {
	c.RLock()
	defer c.RUnlock()
	return c.TransactionName
}

// GetSQLSanitize returns the SQL sanitization level.
//
// The meaning of each level:
// 0 - disable SQL sanitizing (the default).
// 1 - enable SQL sanitizing and attempt to automatically determine which
// quoting form to use.
// 2 - enable SQL sanitizing and force dropping double quoted characters.
// 4 - enable SQL sanitizing and force retaining double quoted character.
func (c *Config) GetSQLSanitize() int {
	c.RLock()
	defer c.RUnlock()
	return c.SQLSanitize
}
