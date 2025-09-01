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

package exporter

import (
	"context"
	"fmt"
	"strings"

	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/solarwinds/apm-go/internal/entryspans"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/reporter"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/txn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type exporter struct {
	r reporter.Reporter
}

func (e *exporter) exportSpan(_ context.Context, s sdktrace.ReadOnlySpan) {
	evt := reporter.CreateEntryEvent(s.SpanContext(), s.StartTime(), s.Parent())
	e.setSpanLevelAoAttributes(evt, s)
	evt.AddKVs([]attribute.KeyValue{
		attribute.String("Language", constants.Go),
		attribute.String("otel.scope.name", s.InstrumentationScope().Name),
		attribute.String("otel.scope.version", s.InstrumentationScope().Version),
	})
	if entryspans.IsEntrySpan(s) {
		evt.AddKV(attribute.String("TransactionName", txn.GetTransactionName(s)))
	}
	if s.Status().Code != codes.Unset {
		if s.Status().Code == codes.Ok {
			evt.AddKV(semconv.OTelStatusCodeOk)
		} else {
			evt.AddKV(semconv.OTelStatusCodeError)
		}
		if s.Status().Description != "" {
			evt.AddKV(semconv.OTelStatusDescriptionKey.String(s.Status().Description))
		}
	}
	evt.AddKVs(s.Attributes())

	if err := e.r.ReportEvent(evt); err != nil {
		log.Warning("cannot send entry event", err)
		return
	}

	for _, otEvt := range s.Events() {
		evt := reporter.EventFromOtelEvent(s.SpanContext(), otEvt)
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
		if err := e.r.ReportEvent(evt); err != nil {
			log.Warningf("could not send %s event: %s", s.Name(), err)
			continue
		}
	}

	evt = reporter.CreateExitEvent(s.SpanContext(), s.EndTime())
	e.setSpanLevelAoAttributes(evt, s)
	if err := e.r.ReportEvent(evt); err != nil {
		log.Warning("cannot send exit event", err)
		return
	}
}

func (e *exporter) setSpanLevelAoAttributes(evt reporter.Event, s sdktrace.ReadOnlySpan) {
	layer := fmt.Sprintf("%s:%s", strings.ToUpper(s.SpanKind().String()), s.Name())
	evt.SetLayer(layer)
	evt.AddKVs([]attribute.KeyValue{
		attribute.String("sw.span_name", s.Name()),
		attribute.String("sw.span_kind", s.SpanKind().String()),
	})
}

func (e *exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.r.WaitForReady(ctx)
	for _, s := range spans {
		e.exportSpan(ctx, s)
	}
	return nil
}

func (e *exporter) Shutdown(ctx context.Context) error {
	return e.r.Shutdown(ctx)
}

func NewAoExporter(r reporter.Reporter) sdktrace.SpanExporter {
	return &exporter{
		r: r,
	}
}
