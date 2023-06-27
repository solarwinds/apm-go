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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics"
)

// MetricOptions is a struct for the optional parameters of a measurement.
type MetricOptions = metrics.MetricOptions

const (
	// MaxTagsCount is the maximum number of tags allowed.
	MaxTagsCount = metrics.MaxTagsCount
)

// The measurements submission errors
var (
	// ErrExceedsTagsCountLimit indicates the count of tags exceeds the limit
	ErrExceedsTagsCountLimit = metrics.ErrExceedsTagsCountLimit
	// ErrExceedsMetricsCountLimit indicates there are too many distinct measurements in a flush cycle.
	ErrExceedsMetricsCountLimit = metrics.ErrExceedsMetricsCountLimit
	// ErrMetricsWithNonPositiveCount indicates the count is negative or zero
	ErrMetricsWithNonPositiveCount = metrics.ErrMetricsWithNonPositiveCount
)

// SummaryMetric submits a summary type measurement to the reporter. The measurements
// will be collected in the background and reported periodically.
func SummaryMetric(name string, value float64, opts MetricOptions) error {
	return metrics.CustomMetrics.Summary(name, value, opts)
}

// IncrementMetric submits a incremental measurement to the reporter. The measurements
// will be collected in the background and reported periodically.
func IncrementMetric(name string, opts MetricOptions) error {
	return metrics.CustomMetrics.Increment(name, opts)
}
