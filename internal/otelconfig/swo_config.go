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
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"gopkg.in/yaml.v3"
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

	sw := root.InstrumentationDevelopment.Go.Solarwinds
	return solarwindsOtelConfig{
		Collector:           sw.Collector,
		SettingsURL:         sw.SettingsURL,
		ServiceKey:          sw.ServiceKey,
		TrustedPath:         sw.TrustedPath,
		Sampling:            sw.Sampling,
		PrependDomain:       sw.PrependDomain,
		HostAlias:           sw.HostAlias,
		Precision:           sw.Precision,
		SQLSanitize:         sw.SQLSanitize,
		TransactionSettings: sw.TransactionSettings,
		Enabled:             sw.Enabled,
		Ec2MetadataTimeout:  sw.Ec2MetadataTimeout,
		DebugLevel:          sw.DebugLevel,
		TriggerTrace:        sw.TriggerTrace,
		Proxy:               sw.Proxy,
		ProxyCertPath:       sw.ProxyCertPath,
		RuntimeMetrics:      sw.RuntimeMetrics,
		ReportQueryString:   sw.ReportQueryString,
		TokenBucketCap:      sw.TokenBucketCap,
		TokenBucketRate:     sw.TokenBucketRate,
		TransactionName:     sw.TransactionName,
	}, nil
}

func buildSolarwindsConfigOptions(solarwindsCfg solarwindsOtelConfig) []config.Option {
	var opts []config.Option
	if solarwindsCfg.Collector != nil {
		opts = append(opts, config.WithCollector(strings.TrimSpace(*solarwindsCfg.Collector)))
	}
	if solarwindsCfg.ServiceKey != nil {
		opts = append(opts, config.WithServiceKey(strings.TrimSpace(*solarwindsCfg.ServiceKey)))
	}
	if solarwindsCfg.SettingsURL != nil {
		v := strings.TrimSpace(*solarwindsCfg.SettingsURL)
		opts = append(opts, func(c *config.Config) { c.SettingsURL = v })
	}
	if solarwindsCfg.TrustedPath != nil {
		v := strings.TrimSpace(*solarwindsCfg.TrustedPath)
		opts = append(opts, func(c *config.Config) { c.TrustedPath = v })
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
		v := *solarwindsCfg.PrependDomain
		opts = append(opts, func(c *config.Config) { c.PrependDomain = v })
	}
	if solarwindsCfg.HostAlias != nil {
		v := strings.TrimSpace(*solarwindsCfg.HostAlias)
		opts = append(opts, func(c *config.Config) { c.HostAlias = v })
	}
	if solarwindsCfg.Precision != nil {
		v := *solarwindsCfg.Precision
		opts = append(opts, func(c *config.Config) { c.Precision = v })
	}
	if solarwindsCfg.SQLSanitize != nil {
		v := *solarwindsCfg.SQLSanitize
		opts = append(opts, func(c *config.Config) { c.SQLSanitize = v })
	}
	if solarwindsCfg.TransactionSettings != nil {
		transactionSettings := make([]config.TransactionFilter, len(*solarwindsCfg.TransactionSettings))
		copy(transactionSettings, *solarwindsCfg.TransactionSettings)
		opts = append(opts, func(c *config.Config) { c.TransactionSettings = transactionSettings })
	}
	if solarwindsCfg.Enabled != nil {
		v := *solarwindsCfg.Enabled
		opts = append(opts, func(c *config.Config) { c.Enabled = v })
	}
	if solarwindsCfg.Ec2MetadataTimeout != nil {
		v := *solarwindsCfg.Ec2MetadataTimeout
		opts = append(opts, func(c *config.Config) { c.Ec2MetadataTimeout = v })
	}
	if solarwindsCfg.DebugLevel != nil {
		v := strings.TrimSpace(*solarwindsCfg.DebugLevel)
		opts = append(opts, func(c *config.Config) { c.DebugLevel = v })
	}
	if solarwindsCfg.TriggerTrace != nil {
		v := *solarwindsCfg.TriggerTrace
		opts = append(opts, func(c *config.Config) { c.TriggerTrace = v })
	}
	if solarwindsCfg.Proxy != nil {
		v := strings.TrimSpace(*solarwindsCfg.Proxy)
		opts = append(opts, func(c *config.Config) { c.Proxy = v })
	}
	if solarwindsCfg.ProxyCertPath != nil {
		v := strings.TrimSpace(*solarwindsCfg.ProxyCertPath)
		opts = append(opts, func(c *config.Config) { c.ProxyCertPath = v })
	}
	if solarwindsCfg.RuntimeMetrics != nil {
		opts = append(opts, config.WithRuntimeMetrics(*solarwindsCfg.RuntimeMetrics))
	}
	if solarwindsCfg.ReportQueryString != nil {
		v := *solarwindsCfg.ReportQueryString
		opts = append(opts, func(c *config.Config) { c.ReportQueryString = v })
	}
	if solarwindsCfg.TokenBucketCap != nil {
		v := *solarwindsCfg.TokenBucketCap
		opts = append(opts, func(c *config.Config) { c.TokenBucketCap = v })
	}
	if solarwindsCfg.TokenBucketRate != nil {
		v := *solarwindsCfg.TokenBucketRate
		opts = append(opts, func(c *config.Config) { c.TokenBucketRate = v })
	}
	if solarwindsCfg.TransactionName != nil {
		v := strings.TrimSpace(*solarwindsCfg.TransactionName)
		opts = append(opts, func(c *config.Config) { c.TransactionName = v })
	}
	return opts
}
