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
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/constants"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/entryspans"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type exporter struct {
}

func exportSpan(_ context.Context, s sdktrace.ReadOnlySpan) {
	evt, err := reporter.CreateEntryEvent(s.SpanContext(), s.StartTime(), s.Parent())
	if err != nil {
		log.Warning("could not create entry event", err)
		return
	}
	layer := fmt.Sprintf("%s:%s", s.SpanKind().String(), s.Name())
	evt.SetLayer(layer)
	evt.AddKVs([]attribute.KeyValue{
		attribute.String("sw.span_name", s.Name()),
		attribute.String("sw.span_kind", s.SpanKind().String()),
		attribute.String("Language", constants.Go),
		attribute.String("otel.scope.name", s.InstrumentationScope().Name),
		attribute.String("otel.scope.version", s.InstrumentationScope().Version),
	})
	if entryspans.IsEntrySpan(s) {
		evt.AddKV(attribute.String("TransactionName", utils.GetTransactionName(s)))
		// We MUST clear the entry span here. The SpanProcessor only clears entry spans when they are `RecordOnly`
		if err := entryspans.Delete(s); err != nil {
			log.Warningf(
				"could not delete entry span for trace-span %s-%s",
				s.SpanContext().TraceID(),
				s.SpanContext().SpanID(),
			)
		}
	}
	evt.AddKVs(s.Attributes())

	err = reporter.ReportEvent(evt)
	if err != nil {
		log.Warning("cannot send entry event", err)
		return
	}

	for _, otEvt := range s.Events() {
		evt, err := reporter.EventFromOtelEvent(s.SpanContext(), otEvt)
		if err != nil {
			log.Warningf("could not create %s event: %s", s.Name(), err)
			continue
		}
		if otEvt.Name == semconv.ExceptionEventName {
			set := attribute.NewSet(otEvt.Attributes...)
			if v, ok := set.Value(semconv.ExceptionMessageKey); ok {
				evt.AddKV(attribute.String("ErrorMsg", v.AsString()))
			}
			if v, ok := set.Value(semconv.ExceptionTypeKey); ok {
				evt.AddKV(attribute.String("ErrorClass", v.AsString()))
			}
			if v, ok := set.Value(semconv.ExceptionStacktraceKey); ok {
				evt.AddKV(attribute.String("Backtrace", v.AsString()))
			}
		}
		evt.AddKVs(otEvt.Attributes)
		err = reporter.ReportEvent(evt)
		if err != nil {
			log.Warningf("could not send %s event: %s", s.Name(), err)
			continue
		}
	}

	evt, err = reporter.CreateExitEvent(s.SpanContext(), s.EndTime())
	if err != nil {
		log.Warning("could not create exit event", err)
		return
	}
	evt.AddKV(attribute.String(constants.Layer, layer))
	err = reporter.ReportEvent(evt)
	if err != nil {
		log.Warning("cannot send exit event", err)
		return
	}

}

func (e *exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	WaitForReady(ctx)
	for _, s := range spans {
		exportSpan(ctx, s)
	}
	return nil
}

func (e *exporter) Shutdown(ctx context.Context) error {
	Shutdown(ctx)
	return nil
}

func NewExporter() sdktrace.SpanExporter {
	return &exporter{}
}
