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
	"context"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var _ sdktrace.SpanProcessor = &SolarWindsInboundMetricsSpanProcessor{}

type SolarWindsInboundMetricsSpanProcessor struct{}

func (s *SolarWindsInboundMetricsSpanProcessor) OnStart(parent context.Context, span sdktrace.ReadWriteSpan) {
	//TODO implement me
	panic("implement me")
}

func (s *SolarWindsInboundMetricsSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	parent := span.Parent()
	if parent.IsValid() && !parent.IsRemote() {
		return
	}
	metrics.RecordSpan(span)
}

func (s *SolarWindsInboundMetricsSpanProcessor) Shutdown(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s *SolarWindsInboundMetricsSpanProcessor) ForceFlush(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}
