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
	traceID := s.SpanContext().TraceID()
	spanID := s.SpanContext().SpanID()
	xTraceID := getXTraceID(traceID[:], spanID[:])

	startOverrides := Overrides{
		ExplicitTS:    s.StartTime(),
		ExplicitMdStr: xTraceID,
	}

	endOverrides := Overrides{
		ExplicitTS: s.EndTime(),
	}

	kvs := extractKvs(s)

	infoEvents := extractInfoEvents(s)

	if s.Parent().IsValid() { // this is a child span, not a start of a trace but rather a continuation of an existing one
		parentSpanID := s.Parent().SpanID()
		parentXTraceID := getXTraceID(traceID[:], parentSpanID[:])
		traceContext := FromXTraceIDContext(ctx, parentXTraceID)
		apmSpan, _ := BeginSpanWithOverrides(traceContext, s.Name(), SpanOptions{}, startOverrides)

		// report otel Span Events as SolarWinds Observability Info KVs
		for _, infoEventKvs := range infoEvents {
			apmSpan.InfoWithOverrides(Overrides{ExplicitTS: s.StartTime()}, SpanOptions{}, infoEventKvs...)
		}
		apmSpan.EndWithOverrides(endOverrides, kvs...)
	} else { // no parent means this is the beginning of the trace (root span)
		trace := NewTraceWithOverrides(s.Name(), startOverrides, nil)
		trace.SetStartTime(s.StartTime()) //this is for histogram only

		// report otel Span Events as SolarWinds Observability Info KVs
		for _, infoEventKvs := range infoEvents {
			trace.InfoWithOverrides(Overrides{ExplicitTS: s.StartTime()}, SpanOptions{}, infoEventKvs...)
		}
		trace.EndWithOverrides(endOverrides, kvs...)
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
