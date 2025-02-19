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

package metrics

import (
	"errors"
	"strconv"
	"time"

	"github.com/solarwinds/apm-go/internal/bson"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/txn"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	trace2 "go.opentelemetry.io/otel/trace"
)

type registry struct {
	apmHistograms *histograms
	apmMetrics    *measurements
	customMetrics *measurements
	isAppoptics   bool
}

var _ LegacyRegistry = &registry{}

func NewLegacyRegistry(isAppoptics bool) LegacyRegistry {
	return &registry{
		apmHistograms: &histograms{
			histograms: make(map[string]*histogram),
			precision:  getPrecision(),
		},
		apmMetrics:    newMeasurements(false, metricsTransactionsMaxDefault),
		customMetrics: newMeasurements(true, metricsCustomMetricsMaxDefault),
		isAppoptics:   isAppoptics,
	}
}

type MetricRegistry interface {
	RecordSpan(span trace.ReadOnlySpan)
}

type LegacyRegistry interface {
	MetricRegistry
	BuildBuiltinMetricsMessage(flushInterval int32, qs *EventQueueStats,
		rcs *RateCountSummary, runtimeMetrics bool) []byte
	BuildCustomMetricsMessage(flushInterval int32) []byte
	ApmMetricsCap() int32
	SetApmMetricsCap(int32)
	CustomMetricsCap() int32
	SetCustomMetricsCap(int32)
}

// BuildCustomMetricsMessage creates and encodes the custom metrics message.
func (r *registry) BuildCustomMetricsMessage(flushInterval int32) []byte {
	m := r.customMetrics.CopyAndReset(flushInterval)
	if m == nil {
		return nil
	}
	bbuf := bson.NewBuffer()
	if m.isCustom {
		bbuf.AppendBool("isCustom", m.isCustom)
	}

	appendHostId(bbuf)
	bbuf.AppendInt32("MetricsFlushInterval", m.flushInterval)

	bbuf.AppendInt64("Timestamp_u", time.Now().UnixNano()/1000)

	start := bbuf.AppendStartArray("measurements")
	index := 0

	for _, measurement := range m.m {
		addMeasurementToBSON(bbuf, &index, measurement)
	}

	bbuf.AppendFinishObject(start)

	bbuf.Finish()
	return bbuf.GetBuf()
}

// BuildBuiltinMetricsMessage generates a metrics message in BSON format with all the currently available values
// metricsFlushInterval	current metrics flush interval
//
// return				metrics message in BSON format
func (r *registry) BuildBuiltinMetricsMessage(flushInterval int32, qs *EventQueueStats,
	rcs *RateCountSummary, runtimeMetrics bool) []byte {
	var m = r.apmMetrics.CopyAndReset(flushInterval)
	if m == nil {
		return nil
	}

	bbuf := bson.NewBuffer()

	appendHostId(bbuf)
	bbuf.AppendInt32("MetricsFlushInterval", flushInterval)

	bbuf.AppendInt64("Timestamp_u", time.Now().UnixNano()/1000)

	// measurements
	// ==========================================
	start := bbuf.AppendStartArray("measurements")
	index := 0

	// request counters
	addRequestCounters(bbuf, &index, rcs)

	// Queue states
	if qs != nil {
		addMetricsValue(bbuf, &index, "NumSent", qs.numSent)
		addMetricsValue(bbuf, &index, "NumOverflowed", qs.numOverflowed)
		addMetricsValue(bbuf, &index, "NumFailed", qs.numFailed)
		addMetricsValue(bbuf, &index, "TotalEvents", qs.totalEvents)
		addMetricsValue(bbuf, &index, "QueueLargest", qs.queueLargest)
	}

	addHostMetrics(bbuf, &index)

	if runtimeMetrics {
		// runtime stats
		addRuntimeMetrics(bbuf, &index)
	}

	for _, measurement := range m.m {
		addMeasurementToBSON(bbuf, &index, measurement)
	}

	bbuf.AppendFinishObject(start)
	// ==========================================

	// histograms
	// ==========================================
	start = bbuf.AppendStartArray("histograms")
	index = 0

	r.apmHistograms.lock.Lock()

	for _, h := range r.apmHistograms.histograms {
		addHistogramToBSON(bbuf, &index, h)
	}
	r.apmHistograms.histograms = make(map[string]*histogram) // clear histograms

	r.apmHistograms.lock.Unlock()
	bbuf.AppendFinishObject(start)
	// ==========================================

	if m.txnMap.isOverflowed() {
		bbuf.AppendBool("TransactionNameOverflow", true)
	}

	bbuf.Finish()
	return bbuf.GetBuf()
}

func (r *registry) RecordSpan(span trace.ReadOnlySpan) {
	method := ""
	status := int64(0)
	isError := span.Status().Code == codes.Error
	attrs := span.Attributes()
	swoTags := make(map[string]string)
	httpRoute := ""
	for _, attr := range attrs {
		if attr.Key == semconv.HTTPMethodKey {
			method = attr.Value.AsString()
		} else if attr.Key == semconv.HTTPStatusCodeKey {
			status = attr.Value.AsInt64()
		} else if attr.Key == semconv.HTTPRouteKey {
			httpRoute = attr.Value.AsString()
		}
	}
	isHttp := span.SpanKind() == trace2.SpanKindServer && method != ""

	if isHttp {
		if status > 0 {
			swoTags["http.status_code"] = strconv.FormatInt(status, 10)
		}
		swoTags["http.method"] = method
	}

	swoTags["sw.is_error"] = strconv.FormatBool(isError)
	txnName := txn.GetTransactionName(span)
	swoTags["sw.transaction"] = txnName

	duration := span.EndTime().Sub(span.StartTime())
	s := &HTTPSpanMessage{
		BaseSpanMessage: BaseSpanMessage{Duration: duration, HasError: isError},
		Transaction:     txnName,
		Path:            httpRoute,
		Status:          int(status),
		Host:            "", // intentionally not set
		Method:          method,
	}

	var tagsList []map[string]string
	var metricName string
	if !r.isAppoptics {
		tagsList = []map[string]string{swoTags}
		metricName = responseTime
	} else {
		tagsList = s.appOpticsTagsList()
		metricName = transactionResponseTime
	}

	r.apmHistograms.recordHistogram("", duration)
	if err := s.processMeasurements(metricName, tagsList, r.apmMetrics); errors.Is(err, ErrExceedsMetricsCountLimit) {
		if r.isAppoptics {
			s.Transaction = OtherTransactionName
			tagsList = s.appOpticsTagsList()
		} else {
			tagsList[0]["sw.transaction"] = OtherTransactionName
		}
		err := s.processMeasurements(metricName, tagsList, r.apmMetrics)
		// This should never happen since the only failure case _should_ be ErrExceedsMetricsCountLimit
		// which is handled above, and the reason we retry here.
		if err != nil {
			log.Errorf("Failed to process messages", err)
		}
	} else {
		// We didn't hit ErrExceedsMetricsCountLimit
		r.apmHistograms.recordHistogram(txnName, duration)
	}

}

func (r *registry) ApmMetricsCap() int32 {
	return r.apmMetrics.Cap()
}

func (r *registry) SetApmMetricsCap(cap int32) {
	r.apmMetrics.SetCap(cap)
}

func (r *registry) CustomMetricsCap() int32 {
	return r.customMetrics.Cap()
}

func (r *registry) SetCustomMetricsCap(cap int32) {
	r.customMetrics.SetCap(cap)
}
