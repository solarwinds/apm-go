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
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type exporter struct {
}

func exportSpan(_ context.Context, s sdktrace.ReadOnlySpan) {
	evt, err := reporter.CreateEntry(s.SpanContext(), s.StartTime(), s.Parent())
	if err != nil {
		log.Warning("could not create entry event", err)
		return
	}
	layer := fmt.Sprintf("%s:%s", s.SpanKind().String(), s.Name())
	evt.AddString("Layer", layer)
	evt.AddString("sw.span_name", s.Name())
	evt.AddString("sw.span_kind", s.SpanKind().String())
	evt.AddString("Language", "Go")
	evt.AddString("otel.scope.name", s.InstrumentationScope().Name)
	evt.AddString("otel.scope.version", s.InstrumentationScope().Version)
	if !s.Parent().IsValid() || s.Parent().IsRemote() {
		// root span only
		evt.AddString("TransactionName", utils.GetTransactionName(s.Name(), s.Attributes()))
	}
	evt.AddAttributes(s.Attributes())

	err = reporter.SendReport(evt)
	if err != nil {
		log.Warning("cannot send entry event", err)
		return
	}

	for _, otEvt := range s.Events() {
		evt, err = reporter.CreateInfoEvent(s.SpanContext(), otEvt.Time)
		if err != nil {
			log.Warning("could not create info event", err)
			continue
		}
		evt.AddAttributes(otEvt.Attributes)
		err = reporter.SendReport(evt)
		if err != nil {
			log.Warning("could not send info event", err)
			continue
		}
	}

	evt, err = reporter.CreateExit(s.SpanContext(), s.EndTime())
	if err != nil {
		log.Warning("could not create exit event", err)
		return
	}
	evt.AddString("Layer", layer)
	err = reporter.SendReport(evt)
	if err != nil {
		log.Warning("cannot sent exit event", err)
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
