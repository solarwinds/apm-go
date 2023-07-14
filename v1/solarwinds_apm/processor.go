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

var _ sdktrace.SpanProcessor = &InboundMetricsProcessor{}

var recordFunc = metrics.RecordSpan

type InboundMetricsProcessor struct {
	isAppoptics bool
}

func (s *InboundMetricsProcessor) OnStart(context.Context, sdktrace.ReadWriteSpan) {
}

func (s *InboundMetricsProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	parent := span.Parent()
	if parent.IsValid() && !parent.IsRemote() {
		return
	}
	recordFunc(span, s.isAppoptics)
}

func (s *InboundMetricsProcessor) Shutdown(context.Context) error {
	return nil
}

func (s *InboundMetricsProcessor) ForceFlush(context.Context) error {
	return nil
}
