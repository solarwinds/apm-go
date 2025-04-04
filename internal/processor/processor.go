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

	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/txn"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewInboundMetricsSpanProcessor(registry metrics.MetricRegistry) sdktrace.SpanProcessor {
	return &inboundMetricsSpanProcessor{
		registry: registry,
	}
}

var _ sdktrace.SpanProcessor = &inboundMetricsSpanProcessor{}

type inboundMetricsSpanProcessor struct {
	registry metrics.MetricRegistry
}

func (s *inboundMetricsSpanProcessor) OnStart(_ context.Context, span sdktrace.ReadWriteSpan) {
	if entryspans.IsEntrySpan(span) {
		// Set default transaction name value. It can be overwritten later using SetTransactionName API.
		span.SetAttributes(attribute.String(constants.SwTransactionNameAttribute, txn.GetTransactionName(span.(sdktrace.ReadOnlySpan))))

		if err := entryspans.Push(span); err != nil {
			// The only error here should be if it's not an entry span, and we've guarded against that,
			// so it's safe to log the error and move on
			log.Warningf("could not push entry span: %s", err)
		}
	}
}

func clearEntrySpan(span sdktrace.ReadOnlySpan) {
	if err := entryspans.Delete(span); err != nil {
		log.Warningf("could not delete entry span for trace-span %s-%s",
			span.SpanContext().TraceID(), span.SpanContext().SpanID())
	}
}

func (s *inboundMetricsSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	if entryspans.IsEntrySpan(span) {
		s.registry.RecordSpan(span)
		clearEntrySpan(span)
	}
}

func (s *inboundMetricsSpanProcessor) Shutdown(context.Context) error {
	return nil
}

func (s *inboundMetricsSpanProcessor) ForceFlush(context.Context) error {
	return nil
}
