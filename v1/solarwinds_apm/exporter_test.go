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
	"go.opentelemetry.io/otel/codes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	ot "go.opentelemetry.io/otel/trace"
)

var tnNameKey = attribute.Key("TransactionName")

func setup() (ot.Tracer, func()) {
	tp := trace.NewTracerProvider(
		trace.WithBatcher(NewDummyExporter()),
		trace.WithSampler(NewDummySampler()),
	)
	otel.SetTracerProvider(tp)
	tr := otel.Tracer("foo123", ot.WithInstrumentationVersion("123"), ot.WithSchemaURL("https://www.schema.url/foo123"))

	return tr, func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
		}
	}
}

type DummySampler struct{}

func (ds *DummySampler) ShouldSample(parameters trace.SamplingParameters) trace.SamplingResult {
	return trace.SamplingResult{
		Decision: trace.RecordAndSample,
	}
}

func (ds *DummySampler) Description() string {
	return "Dummy Sampler"
}

func NewDummySampler() trace.Sampler {
	return &DummySampler{}
}

type DummyExporter struct{}

func NewDummyExporter() *DummyExporter {
	return &DummyExporter{}
}

func (de *DummyExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return nil
}

func (de *DummyExporter) Shutdown(ctx context.Context) error {
	return nil
}

func Test_extractKvs_Basic(t *testing.T) {
	tr, teardown := setup()
	defer teardown()
	_, sp := tr.Start(context.Background(), "ROOT SPAN NAME aaaa")
	spanName := "span name"
	sp.SetName(spanName)

	priority_val := "high"
	priority := attribute.Key("some.priority")
	answers := attribute.Key("some.answers-bool-slice")

	boolSlice := []bool{true, false}
	sp.SetAttributes(priority.String(priority_val), answers.BoolSlice(boolSlice))
	sp.SetStatus(codes.Error, "uh oh, problem!") // when the status code is an error, a description "uh oh, problem!" will be inserted into the span as well
	sp.End()

	kvs := extractKvs(sp.(trace.ReadOnlySpan))

	require.Equal(t, []interface{}{
		string(priority), priority_val,
		string(answers), boolSlice,
		string(tnNameKey), spanName,
	}, kvs)
}

func Test_extractKvs_Empty(t *testing.T) {
	tr, teardown := setup()
	defer teardown()
	_, sp := tr.Start(context.Background(), "ROOT SPAN NAME aaaa")
	spanName := "span name"
	sp.SetName(spanName)

	attrs := make([]attribute.KeyValue, 0)
	sp.SetAttributes(attrs...)

	kvs := extractKvs(sp.(trace.ReadOnlySpan))

	require.Equal(t, []interface{}{
		string(tnNameKey), spanName,
	}, kvs)
}

func Test_extractInfoEvents(t *testing.T) {
	tr, teardown := setup()
	defer teardown()
	_, sp := tr.Start(context.Background(), "ROOT SPAN NAME aaaa")
	sp.SetName("span name")

	float64attr := attribute.Key("some.float64-slice")
	stringslice := attribute.Key("some.string-slice")
	sp.SetAttributes(float64attr.Float64Slice([]float64{2.3490, 3.14159, 0.49581}), stringslice.StringSlice([]string{"string1", "string2", "string3"}))

	sp.AddEvent("auth", ot.WithAttributes(attribute.String("username", "joe"), attribute.Int("uid", 100)))
	sp.AddEvent("buy", ot.WithAttributes(attribute.String("product", "iPhone"), attribute.Float64("price", 799.99)))
	sp.AddEvent("unsubscribe", ot.WithAttributes(attribute.String("mailing-list-id", "list1"), attribute.Bool("eula-read", true)))

	slInt64 := []int64{-1337, 30, 2, 30000, 45}
	sp.AddEvent("test-int64-slice-event", ot.WithAttributes(attribute.Int64Slice("int64-slice-key", slInt64)))
	slString := []string{"s1", "s2", "s3", "s4"}
	sp.AddEvent("test-string-slice-event", ot.WithAttributes(attribute.StringSlice("string-slice-key", slString)))
	slBool := []bool{true, false, false, true}
	sp.AddEvent("test-bool-slice-event", ot.WithAttributes(attribute.BoolSlice("bool-slice-key", slBool)))
	slFloat64 := []float64{-3.14159, 300.30409, 2, 2.0, 2.001}
	sp.AddEvent("test-float64-slice-event", ot.WithAttributes(attribute.Float64Slice("float64-slice-key", slFloat64)))
	sp.SetStatus(2, "all good!") // description is ignored if the status is not an error
	sp.End()

	kvs := extractKvs(sp.(trace.ReadOnlySpan))
	require.Equal(t, 6, len(kvs), "kvs length mismatch: ", kvs)

	infoEvents := extractInfoEvents(sp.(trace.ReadOnlySpan))
	require.Equal(t, 7, len(infoEvents), "infoEvents length mismatch")

	require.Equal(t, []interface{}{"username", "joe", "uid", int64(100)}, infoEvents[0])
	require.Equal(t, []interface{}{"product", "iPhone", "price", float64(799.99)}, infoEvents[1])
	require.Equal(t, []interface{}{"mailing-list-id", "list1", "eula-read", true}, infoEvents[2])
	require.Equal(t, []interface{}{"int64-slice-key", []int64{-1337, 30, 2, 30000, 45}}, infoEvents[3])
	require.Equal(t, []interface{}{"string-slice-key", []string{"s1", "s2", "s3", "s4"}}, infoEvents[4])
	require.Equal(t, []interface{}{"bool-slice-key", []bool{true, false, false, true}}, infoEvents[5])
	require.Equal(t, []interface{}{"float64-slice-key", []float64{-3.14159, 300.30409, 2, 2, 2.001}}, infoEvents[6])
}

func Test_getXTraceID(t *testing.T) {
	var traceID ot.TraceID
	var spanID ot.SpanID

	traceID = [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID = [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	xTraceID := getXTraceID(traceID[:], spanID[:])
	//2B 0102030405060708090A0B0C0D0E0F10 00000000 0102030405060708 01
	expectedXTraceID := strings.ToUpper(xtraceVersionHeader + hex.EncodeToString(traceID[:]) + "00000000" + hex.EncodeToString(spanID[:]) + sampledFlags)
	require.Equal(t, expectedXTraceID, xTraceID, "xTraceID should be equal")
}
