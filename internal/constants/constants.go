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

package constants

const (

	// Other

	Edge       = "Edge"
	Go         = "Go"
	Label      = "Label"
	Layer      = "Layer"
	TraceState = "tracestate"

	// Label strings

	EntryLabel   = "entry"
	ErrorLabel   = "error"
	ExitLabel    = "exit"
	InfoLabel    = "info"
	UnknownLabel = "UNKNOWN"
)

const (
	KvSignatureKey                      = "SignatureKey"
	KvBucketCapacity                    = "BucketCapacity"
	KvBucketRate                        = "BucketRate"
	KvTriggerTraceRelaxedBucketCapacity = "TriggerRelaxedBucketCapacity"
	KvTriggerTraceRelaxedBucketRate     = "TriggerRelaxedBucketRate"
	KvTriggerTraceStrictBucketCapacity  = "TriggerStrictBucketCapacity"
	KvTriggerTraceStrictBucketRate      = "TriggerStrictBucketRate"
	KvMetricsFlushInterval              = "MetricsFlushInterval"
	KvEventsFlushInterval               = "EventsFlushInterval"
	KvMaxTransactions                   = "MaxTransactions"
	KvMaxCustomMetrics                  = "MaxCustomMetrics"
)

const (
	SwTransactionNameAttribute = "sw.transaction"
	UamsClientIdAttribute      = "sw.uams.client.id"
)
