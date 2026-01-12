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

package metrics

// RateCountSummary is used to merge RateCounts from multiple token buckets
type RateCountSummary struct {
	Requested, Traced, Limited, TtTraced, Sampled, Through int64
}

type HostMetrics interface {
	getTotalRAM() (uint64, bool)
	getFreeRAM() (uint64, bool)
	getSystemLoad1() (float64, bool)
}
