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

package solarwinds_apm

import (
	"time"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
)

type Overrides struct {
	ExplicitTS    time.Time
	ExplicitMdStr string
}

// KVMap is a map of additional key-value pairs to report along with the event data provided
// to SolarWinds Observability. Certain key names (such as "Query" or "RemoteHost") are used by SolarWinds Observability to
// provide details about program activity and distinguish between different types of spans.
// Please visit [TODO] for
// details on the key names that SolarWinds Observability looks for.
type KVMap = reporter.KVMap

// ContextOptions is an alias of the reporter's ContextOptions
type ContextOptions = reporter.ContextOptions
