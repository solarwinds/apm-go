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
	"encoding/hex"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/reporter"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"strings"
)

type exporter struct {
}

const (
	xtraceVersionHeader = "2B"
	sampledFlags        = "01"
)

func extractKvs(span sdktrace.ReadOnlySpan) []interface{} {
	var kvs []interface{}
	for _, attributeValue := range span.Attributes() {
		kvs = append(kvs, string(attributeValue.Key), attributeValue.Value.AsInterface())
	}

	if !span.Parent().IsValid() || span.Parent().IsRemote() {
		// root span only
		kvs = append(kvs, "TransactionName", utils.GetTransactionName(span.Name(), span.Attributes()))
	}

	return kvs
}

func extractInfoEvents(span sdktrace.ReadOnlySpan) [][]interface{} {
	events := span.Events()
	kvs := make([][]interface{}, len(events))

	for i, event := range events {
		kvs[i] = make([]interface{}, 0)
		for _, attr := range event.Attributes {
			kvs[i] = append(kvs[i], string(attr.Key), attr.Value.AsInterface())
		}
	}

	return kvs
}

func getXTraceID(traceID []byte, spanID []byte) string {
	taskId := strings.ToUpper(strings.ReplaceAll(fmt.Sprintf("%0-40v", hex.EncodeToString(traceID)), " ", "0"))
	opId := strings.ToUpper(strings.ReplaceAll(fmt.Sprintf("%0-16v", hex.EncodeToString(spanID)), " ", "0"))
	return xtraceVersionHeader + taskId + opId + sampledFlags
}

func exportSpan(ctx context.Context, s sdktrace.ReadOnlySpan) {
	// TODO
	//            for event in span.events:
	//                if event.name == "exception":
	//                    self._report_exception_event(event)
	//                else:
	//                    self._report_info_event(event)
	//
	// TODO add txn name
	// TODO add instrumentation scope
	// TODO add instrumented framework?
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

	for _, kv := range s.Attributes() {
		err := evt.AddKV(string(kv.Key), kv.Value)
		if err != nil {
			log.Warning("could not add KV", kv, err)
			// Continue so we don't completely abandon the event
		}
	}

	err = reporter.SendReport(evt)
	if err != nil {
		log.Warning("cannot sent entry event", err)
		return
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
