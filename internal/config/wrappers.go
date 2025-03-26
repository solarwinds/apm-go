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
	"github.com/solarwinds/apm-go/internal/log"
)

var conf = NewConfig()

// GetCollector is a wrapper to the method of the global config
var GetCollector = conf.GetCollector

// GetServiceKey is a wrapper to the method of the global config
var GetServiceKey = conf.GetServiceKey

// GetTrustedPath is a wrapper to the method of the global config
var GetTrustedPath = conf.GetTrustedPath

// GetTracingMode is a wrapper to the method of the global config
var GetTracingMode = conf.GetTracingMode

// GetSampleRate is a wrapper to the method of the global config
var GetSampleRate = conf.GetSampleRate

// SamplingConfigured is a wrapper to the method of the global config
var SamplingConfigured = conf.SamplingConfigured

// GetPrependDomain is a wrapper to the method of the global config
var GetPrependDomain = conf.GetPrependDomain

// GetHostAlias is a wrapper to the method of the global config
var GetHostAlias = conf.GetHostAlias

// GetPrecision is a wrapper to the method of the global config
var GetPrecision = conf.GetPrecision

// GetEnabled is a wrapper to the method of the global config
var GetEnabled = conf.GetEnabled

// ReporterOpts is a wrapper to the method of the global config
var ReporterOpts = conf.GetReporter

// GetEc2MetadataTimeout is a wrapper to the method of the global config
var GetEc2MetadataTimeout = conf.GetEc2MetadataTimeout

// DebugLevel is a wrapper to the method of the global config
var DebugLevel = conf.GetDebugLevel

// GetTriggerTrace is a wrapper to the method of the global config
var GetTriggerTrace = conf.GetTriggerTrace

// GetProxy is a wrapper to the method of the global config
var GetProxy = conf.GetProxy

// GetProxyCertPath is a wrapper to the method of the global config
var GetProxyCertPath = conf.GetProxyCertPath

// GetRuntimeMetrics is a wrapper to the method of the global config
var GetRuntimeMetrics = conf.GetRuntimeMetrics

var GetTokenBucketCap = conf.GetTokenBucketCap
var GetTokenBucketRate = conf.GetTokenBucketRate
var GetReportQueryString = conf.GetReportQueryString

// GetTransactionFiltering is a wrapper to the method of the global config
var GetTransactionFiltering = conf.GetTransactionFiltering

var GetTransactionName = conf.GetTransactionName

// GetSQLSanitize is a wrapper to method GetSQLSanitize of the global variable config.
var GetSQLSanitize = conf.GetSQLSanitize

var GetServiceNameFromServiceKey = conf.GetServiceNameFromServiceKey

var GetOtelCollector = conf.GetOtelCollector

var GetApiToken = conf.GetApiToken

// Load reads the customized configurations
var Load = conf.Load

var GetDelta = conf.GetDelta

func init() {
	if conf.GetEnabled() {
		log.Warningf("Accepted config items: \n%s", conf.GetDelta())
	}
}
