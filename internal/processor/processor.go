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

package processor

import (
	"context"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/entryspans"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/metrics"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewInboundMetricsSpanProcessor(isAppoptics bool) sdktrace.SpanProcessor {
	return &inboundMetricsSpanProcessor{
		isAppoptics: isAppoptics,
	}
}

var _ sdktrace.SpanProcessor = &inboundMetricsSpanProcessor{}

var recordFunc = metrics.RecordSpan

type inboundMetricsSpanProcessor struct {
	isAppoptics bool
}

func (s *inboundMetricsSpanProcessor) OnStart(_ context.Context, span sdktrace.ReadWriteSpan) {
	if entryspans.IsEntrySpan(span) {
		if err := entryspans.Push(span); err != nil {
			// The only error here should be if it's not an entry span, and we've guarded against that,
			// so it's safe to log the error and move on
			log.Warningf("could not push entry span: %s", err)
		}
	}
}

func maybeClearEntrySpan(span sdktrace.ReadOnlySpan) {
	if span.SpanContext().IsSampled() {
		// Do not clear here. The exporter will need the added context and will
		// clear. If we clear here, the exporter will not see the entry
		// span state.
		return
	}
	// Not sampled; the exporter will not see it, thus we must clear.
	if err := entryspans.Delete(span); err != nil {
		log.Warningf("could not delete entry span for trace-span %s-%s",
			span.SpanContext().TraceID(), span.SpanContext().SpanID())
	}
}

func (s *inboundMetricsSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	if entryspans.IsEntrySpan(span) {
		recordFunc(span, s.isAppoptics)
		maybeClearEntrySpan(span)
	}
}

func (s *inboundMetricsSpanProcessor) Shutdown(context.Context) error {
	return nil
}

func (s *inboundMetricsSpanProcessor) ForceFlush(context.Context) error {
	return nil
}
