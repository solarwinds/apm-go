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
// Package reporter provides a low-level API for creating and reporting events for
// distributed tracing with SolarWinds Observability.

package reporter

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/constants"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/host"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"time"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/bson"
)

type opID [8]byte

// Label is a required event attribute.
type Label int

const (
	LabelUnset Label = iota
	LabelEntry
	LabelExit
	LabelInfo
	LabelError
	LabelSingle
)

func (l Label) AsString() string {
	switch l {
	case LabelEntry:
		return constants.EntryLabel
	case LabelError:
		return constants.ErrorLabel
	case LabelExit:
		return constants.ExitLabel
	case LabelInfo:
		return constants.InfoLabel
	case LabelSingle:
		return constants.SingleLabel
	}
	return constants.UnknownLabel
}

type Event interface {
	AddKV(attribute.KeyValue)
	AddKVs([]attribute.KeyValue)

	SetLabel(Label)
	SetLayer(string)
	SetParent(trace.SpanID)

	ToBson() []byte
}

type event struct {
	taskID trace.TraceID
	opID   [8]byte
	t      time.Time
	kvs    []attribute.KeyValue

	label  Label
	layer  string
	parent trace.SpanID
}

func NewEvent(tid trace.TraceID, oid opID, t time.Time) (Event, error) {
	return &event{
		taskID: tid,
		opID:   oid,
		t:      t,
	}, nil
}

func NewEventWithRandomOpID(tid trace.TraceID, t time.Time) (Event, error) {
	oid := opID{0}
	if _, err := rand.Reader.Read(oid[:]); err != nil {
		return nil, err
	}
	return NewEvent(tid, oid, t)
}

func (e *event) SetLabel(label Label) {
	e.label = label
}

func (e *event) SetLayer(layer string) {
	e.layer = layer
}

func (e *event) SetParent(spanID trace.SpanID) {
	e.parent = spanID
}

func (e *event) AddKV(kv attribute.KeyValue) {
	e.kvs = append(e.kvs, kv)
}

func (e *event) AddKVs(kvs []attribute.KeyValue) {
	e.kvs = append(e.kvs, kvs...)
}

func (e *event) getSwTraceContext() string {
	// For now the version and flags are always 00 and 01, respectively
	return fmt.Sprintf("00-%s-%s-01", e.taskID.String(), hex.EncodeToString(e.opID[:]))
}

func (e *event) getXTrace() string {
	tid := strings.ToUpper(e.taskID.String())
	oid := strings.ToUpper(hex.EncodeToString(e.opID[:]))
	return fmt.Sprintf("2B%s00000000%s01", tid, oid)
}

func (e *event) ToBson() []byte {
	buf := bson.NewBuffer()
	buf.AppendString("sw.trace_context", e.getSwTraceContext())
	buf.AppendString("X-Trace", e.getXTrace())
	buf.AppendInt64("Timestamp_u", e.t.UnixMicro())
	buf.AppendString("Hostname", host.Hostname())
	buf.AppendInt("PID", host.PID())
	buf.AppendString(constants.Label, e.label.AsString())
	if e.layer != "" {
		buf.AppendString(constants.Layer, e.layer)
	}

	if e.parent.IsValid() {
		hx := e.parent.String()
		buf.AppendString(constants.Edge, strings.ToUpper(hx))
		buf.AppendString("sw.parent_span_id", hx)
	}

	for _, kv := range e.kvs {
		if err := buf.AddKV(kv); err != nil {
			log.Warningf("could not add kv", kv, err)
		}
	}
	buf.Finish()
	return buf.GetBuf()
}

func CreateEntry(ctx trace.SpanContext, t time.Time, parent trace.SpanContext) (Event, error) {
	evt, err := NewEvent(ctx.TraceID(), opID(ctx.SpanID()), t)
	if err != nil {
		return nil, err
	}
	if parent.IsValid() {
		evt.SetParent(parent.SpanID())
	}
	evt.SetLabel(LabelEntry)
	return evt, nil
}

func createNonEntryEvent(ctx trace.SpanContext, t time.Time, label Label) (Event, error) {
	evt, err := NewEventWithRandomOpID(ctx.TraceID(), t)
	if err != nil {
		return nil, err
	}
	evt.SetParent(ctx.SpanID())
	evt.SetLabel(label)
	return evt, nil
}

func CreateExit(ctx trace.SpanContext, t time.Time) (Event, error) {
	return createNonEntryEvent(ctx, t, LabelExit)
}

func CreateInfoEvent(ctx trace.SpanContext, t time.Time) (Event, error) {
	return createNonEntryEvent(ctx, t, LabelInfo)
}
