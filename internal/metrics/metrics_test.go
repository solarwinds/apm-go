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
	"bytes"
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/solarwinds/apm-go/internal/bson"
	"github.com/solarwinds/apm-go/internal/hdrhist"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/testutils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/stretchr/testify/assert"
	mbson "gopkg.in/mgo.v2/bson"
)

func bsonToMap(bbuf *bson.Buffer) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	if err := mbson.Unmarshal(bbuf.GetBuf(), m); err != nil {
		return nil, err
	}
	return m, nil
}

func round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func TestAppendIPAddresses(t *testing.T) {
	bbuf := bson.NewBuffer()
	appendIPAddresses(bbuf)
	bbuf.Finish()
	m, err := bsonToMap(bbuf)
	require.NoError(t, err)

	ifaces, _ := host.FilteredIfaces()
	var addresses []string

	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && host.IsPhysicalInterface(iface.Name) {
				addresses = append(addresses, ipnet.IP.String())
			}
		}
	}

	if m["IPAddresses"] != nil {
		bsonIPs := m["IPAddresses"].([]interface{})
		assert.Equal(t, len(bsonIPs), len(addresses))

		for i := 0; i < len(bsonIPs); i++ {
			assert.Equal(t, bsonIPs[i], addresses[i])
		}
	} else {
		assert.Equal(t, 0, len(addresses))
	}
}

func TestAppendMACAddresses(t *testing.T) {
	host.Start()

	bbuf := bson.NewBuffer()
	appendMACAddresses(bbuf, host.CurrentID().MAC())
	bbuf.Finish()
	m, err := bsonToMap(bbuf)
	require.NoError(t, err)

	ifaces, _ := host.FilteredIfaces()
	var macs []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if !host.IsPhysicalInterface(iface.Name) {
			continue
		}
		if mac := iface.HardwareAddr.String(); mac != "" {
			macs = append(macs, iface.HardwareAddr.String())
		}
	}

	if m["MACAddresses"] != nil {
		bsonMACs := m["MACAddresses"].([]interface{})
		assert.Equal(t, len(bsonMACs), len(macs))

		for i := 0; i < len(bsonMACs); i++ {
			assert.Equal(t, bsonMACs[i], macs[i])
		}
	} else {
		assert.Equal(t, 0, len(macs))
	}
}

func TestAddMetricsValue(t *testing.T) {
	index := 0
	bbuf := bson.NewBuffer()
	addMetricsValue(bbuf, &index, "name1", 111)
	addMetricsValue(bbuf, &index, "name2", int64(222))
	addMetricsValue(bbuf, &index, "name3", float32(333.33))
	addMetricsValue(bbuf, &index, "name4", 444.44)
	addMetricsValue(bbuf, &index, "name5", "hello")
	bbuf.Finish()
	m, err := bsonToMap(bbuf)
	require.NoError(t, err)

	assert.NotZero(t, m["0"])
	m2 := m["0"].(map[string]interface{})
	assert.Equal(t, "name1", m2["name"])
	assert.Equal(t, 111, m2["value"])

	assert.NotZero(t, m["1"])
	m2 = m["1"].(map[string]interface{})
	assert.Equal(t, "name2", m2["name"])
	assert.Equal(t, int64(222), m2["value"])

	assert.NotZero(t, m["2"])
	m2 = m["2"].(map[string]interface{})
	assert.Equal(t, "name3", m2["name"])
	f64 := m2["value"].(float64)
	assert.Equal(t, 333.33, round(f64, .5, 2))

	assert.NotZero(t, m["3"])
	m2 = m["3"].(map[string]interface{})
	assert.Equal(t, "name4", m2["name"])
	assert.Equal(t, 444.44, m2["value"])

	assert.NotZero(t, m["4"])
	m2 = m["4"].(map[string]interface{})
	assert.Equal(t, "name5", m2["name"])
	assert.Equal(t, "unknown", m2["value"])
}

func TestGetTransactionFromURL(t *testing.T) {
	type record struct {
		url         string
		transaction string
	}
	var test = []record{
		{
			"/solarwinds/apm-go/blob/metrics/reporter.go#L867",
			"/solarwinds/apm-go",
		},
		{
			"/librato",
			"/librato",
		},
		{
			"",
			"/",
		},
		{
			"/solarwinds/apm-go/blob",
			"/solarwinds/apm-go",
		},
		{
			"/solarwinds/apm-go/blob",
			"/solarwinds/apm-go",
		},
		{
			"http://test.com/solarwinds/apm-go/blob",
			"http://test.com",
		},
		{
			"$%@#%/$%#^*$&/ 1234 4!@ 145412! / 13%1 /14%!$#%^#%& ? 6/``/ ?dfgdf",
			"$%@#%/$%#^*$&/ 1234 4!@ 145412! ",
		},
	}

	for _, r := range test {
		assert.Equal(t, r.transaction, GetTransactionFromPath(r.url), "url: "+r.url)
	}
}

func TestRecordMeasurement(t *testing.T) {
	var me = newMeasurements(false, 100)

	t1 := make(map[string]string)
	t1["t1"] = "tag1"
	t1["t2"] = "tag2"
	err := me.recordWithSoloTags("name1", t1, 111.11, 1, false)
	require.NoError(t, err)
	err = me.recordWithSoloTags("name1", t1, 222, 1, false)
	require.NoError(t, err)
	assert.NotNil(t, me.m["name1&false&t1:tag1&t2:tag2&"])
	m := me.m["name1&false&t1:tag1&t2:tag2&"]
	assert.Equal(t, "tag1", m.Tags["t1"])
	assert.Equal(t, "tag2", m.Tags["t2"])
	assert.Equal(t, 333.11, m.Sum)
	assert.Equal(t, 2, m.Count)
	assert.False(t, m.ReportSum)

	t2 := make(map[string]string)
	t2["t3"] = "tag3"
	err = me.recordWithSoloTags("name2", t2, 123.456, 3, true)
	require.NoError(t, err)
	assert.NotNil(t, me.m["name2&true&t3:tag3&"])
	m = me.m["name2&true&t3:tag3&"]
	assert.Equal(t, "tag3", m.Tags["t3"])
	assert.Equal(t, 123.456, m.Sum)
	assert.Equal(t, 3, m.Count)
	assert.True(t, m.ReportSum)
}

func TestRecordHistogram(t *testing.T) {
	var hi = &histograms{
		histograms: make(map[string]*histogram),
	}

	hi.recordHistogram("", time.Duration(123))
	hi.recordHistogram("", time.Duration(1554))
	assert.NotNil(t, hi.histograms[""])
	h := hi.histograms[""]
	assert.Empty(t, h.tags["TransactionName"])
	encoded, _ := hdrhist.EncodeCompressed(h.hist)
	assert.Equal(t, "HISTFAAAACR42pJpmSzMwMDAxIAKGEHEtclLGOw/QASYmAABAAD//1njBIo=", string(encoded))

	hi.recordHistogram("hist1", time.Duration(453122))
	assert.NotNil(t, hi.histograms["hist1"])
	h = hi.histograms["hist1"]
	assert.Equal(t, "hist1", h.tags["TransactionName"])
	encoded, _ = hdrhist.EncodeCompressed(h.hist)
	assert.Equal(t, "HISTFAAAACR42pJpmSzMwMDAxIAKGEHEtclLGOw/QAQEmQABAAD//1oBBJk=", string(encoded))

	var buf bytes.Buffer
	log.SetOutput(&buf)
	hi.recordHistogram("hist2", time.Duration(4531224545454563))
	log.SetOutput(os.Stderr)
	assert.Contains(t, buf.String(), "Failed to record histogram: value to large")
}

func TestAddMeasurementToBSON(t *testing.T) {
	veryLongTagName := "verylongnameAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	veryLongTagValue := "verylongtagAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	veryLongTagNameTrimmed := veryLongTagName[0:64]
	veryLongTagValueTrimmed := veryLongTagValue[0:255]

	tags1 := make(map[string]string)
	tags1["t1"] = "tag1"
	tags2 := make(map[string]string)
	tags2[veryLongTagName] = veryLongTagValue

	measurement1 := &measurement{
		Name:      "name1",
		Tags:      tags1,
		Count:     45,
		Sum:       592.42,
		ReportSum: false,
	}
	measurement2 := &measurement{
		Name:      "name2",
		Tags:      tags2,
		Count:     777,
		Sum:       6530.3,
		ReportSum: true,
	}

	index := 0
	bbuf := bson.NewBuffer()
	addMeasurementToBSON(bbuf, &index, measurement1)
	addMeasurementToBSON(bbuf, &index, measurement2)
	bbuf.Finish()
	m, err := bsonToMap(bbuf)
	require.NoError(t, err)

	assert.NotZero(t, m["0"])
	m1 := m["0"].(map[string]interface{})
	assert.Equal(t, "name1", m1["name"])
	assert.Equal(t, 45, m1["count"])
	assert.Nil(t, m1["sum"])
	assert.NotZero(t, m1["tags"])
	t1 := m1["tags"].(map[string]interface{})
	assert.Equal(t, "tag1", t1["t1"])

	assert.NotZero(t, m["1"])
	m2 := m["1"].(map[string]interface{})
	assert.Equal(t, "name2", m2["name"])
	assert.Equal(t, 777, m2["count"])
	assert.Equal(t, 6530.3, m2["sum"])
	assert.NotZero(t, m2["tags"])
	t2 := m2["tags"].(map[string]interface{})
	assert.Nil(t, t2[veryLongTagName])
	assert.Equal(t, veryLongTagValueTrimmed, t2[veryLongTagNameTrimmed])
}

func TestAddHistogramToBSON(t *testing.T) {
	veryLongTagName := "verylongnameAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	veryLongTagValue := "verylongtagAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	veryLongTagNameTrimmed := veryLongTagName[0:64]
	veryLongTagValueTrimmed := veryLongTagValue[0:255]

	tags1 := make(map[string]string)
	tags1["t1"] = "tag1"
	tags2 := make(map[string]string)
	tags2[veryLongTagName] = veryLongTagValue

	h1 := &histogram{
		hist: hdrhist.WithConfig(hdrhist.Config{
			LowestDiscernible: 1,
			HighestTrackable:  3600000000,
			SigFigs:           3,
		}),
		tags: tags1,
	}
	h1.hist.Record(34532123)
	h2 := &histogram{
		hist: hdrhist.WithConfig(hdrhist.Config{
			LowestDiscernible: 1,
			HighestTrackable:  3600000000,
			SigFigs:           3,
		}),
		tags: tags2,
	}
	h2.hist.Record(39023)

	index := 0
	bbuf := bson.NewBuffer()
	addHistogramToBSON(bbuf, &index, h1)
	addHistogramToBSON(bbuf, &index, h2)
	bbuf.Finish()
	m, err := bsonToMap(bbuf)
	require.NoError(t, err)

	assert.NotZero(t, m["0"])
	m1 := m["0"].(map[string]interface{})
	assert.Equal(t, "TransactionResponseTime", m1["name"])
	assert.Equal(t, "HISTFAAAACh42pJpmSzMwMDAwgABzFCaEURcm7yEwf4DRGBnAxMTIAAA//9n9AXI", m1["value"])
	assert.NotZero(t, m1["tags"])
	t1 := m1["tags"].(map[string]interface{})
	assert.Equal(t, "tag1", t1["t1"])

	assert.NotZero(t, m["1"])
	m2 := m["1"].(map[string]interface{})
	assert.Equal(t, "TransactionResponseTime", m2["name"])
	assert.Equal(t, "HISTFAAAACZ42pJpmSzMwMDAzAABMJoRRFybvITB/gNEoDWZCRAAAP//YTIFdA==", m2["value"])
	assert.NotZero(t, m2["tags"])
	t2 := m2["tags"].(map[string]interface{})
	assert.Nil(t, t2[veryLongTagName])
	assert.Equal(t, veryLongTagValueTrimmed, t2[veryLongTagNameTrimmed])
}

func TestGenerateMetricsMessage(t *testing.T) {
	reg := NewLegacyRegistry(false).(*registry)
	flushInterval := int32(60)
	bbuf := bson.WithBuf(reg.BuildBuiltinMetricsMessage(flushInterval, &EventQueueStats{},
		&RateCountSummary{
			Requested: 10,
			Sampled:   2,
			Limited:   5,
			Traced:    5,
			Through:   1,
			TtTraced:  3,
		}, true))
	m, err := bsonToMap(bbuf)
	require.NoError(t, err)

	_, ok := m["Hostname"]
	assert.False(t, ok)
	_, ok = m["PID"]
	assert.False(t, ok)
	_, ok = m[""]
	assert.False(t, ok)
	assert.Equal(t, host.Distro(), m["Distro"])
	assert.True(t, m["Timestamp_u"].(int64) > 1509053785684891)

	mts := m["measurements"].([]interface{})

	type testCase struct {
		name  string
		value interface{}
	}

	testCases := []testCase{
		{"RequestCount", int64(10)},
		{"TraceCount", int64(5)},
		{"TokenBucketExhaustionCount", int64(5)},
		{"SampleCount", int64(2)},
		{"ThroughTraceCount", int64(1)},
		{"TriggeredTraceCount", int64(3)},
		{"NumSent", int64(1)},
		{"NumOverflowed", int64(1)},
		{"NumFailed", int64(1)},
		{"TotalEvents", int64(1)},
		{"QueueLargest", int64(1)},
	}
	if runtime.GOOS == "linux" {
		testCases = append(testCases, []testCase{
			{"Load1", float64(1)},
			{"TotalRAM", int64(1)},
			{"FreeRAM", int64(1)},
			{"ProcessRAM", 1},
		}...)
	}
	testCases = append(testCases, []testCase{
		// runtime
		{"trace.go.runtime.NumGoroutine", 1},
		{"trace.go.runtime.NumCgoCall", int64(1)},
		// gc
		{"trace.go.gc.LastGC", int64(1)},
		{"trace.go.gc.NextGC", int64(1)},
		{"trace.go.gc.PauseTotalNs", int64(1)},
		{"trace.go.gc.NumGC", int64(1)},
		{"trace.go.gc.NumForcedGC", int64(1)},
		{"trace.go.gc.GCCPUFraction", float64(1)},
		// memory
		{"trace.go.memory.Alloc", int64(1)},
		{"trace.go.memory.TotalAlloc", int64(1)},
		{"trace.go.memory.Sys", int64(1)},
		{"trace.go.memory.Lookups", int64(1)},
		{"trace.go.memory.Mallocs", int64(1)},
		{"trace.go.memory.Frees", int64(1)},
		{"trace.go.memory.HeapAlloc", int64(1)},
		{"trace.go.memory.HeapSys", int64(1)},
		{"trace.go.memory.HeapIdle", int64(1)},
		{"trace.go.memory.HeapInuse", int64(1)},
		{"trace.go.memory.HeapReleased", int64(1)},
		{"trace.go.memory.HeapObjects", int64(1)},
		{"trace.go.memory.StackInuse", int64(1)},
		{"trace.go.memory.StackSys", int64(1)},
	}...)

	assert.Equal(t, len(testCases), len(mts))

	for i, tc := range testCases {
		assert.Equal(t, tc.name, mts[i].(map[string]interface{})["name"])
		assert.IsType(t, mts[i].(map[string]interface{})["value"], tc.value, tc.name)
		// test the values of the sample rate metrics
		if i < 6 {
			assert.Equal(t, tc.value, mts[i].(map[string]interface{})["value"], tc.name)
		}
	}

	assert.Nil(t, m["TransactionNameOverflow"])

	reg = NewLegacyRegistry(false).(*registry)
	for i := 0; i <= metricsTransactionsMaxDefault; i++ {
		if !reg.apmMetrics.txnMap.isWithinLimit("Transaction-" + strconv.Itoa(i)) {
			break
		}
	}

	m, err = bsonToMap(bson.WithBuf(reg.BuildBuiltinMetricsMessage(flushInterval, &EventQueueStats{},
		&RateCountSummary{}, true)))
	require.NoError(t, err)

	assert.NotNil(t, m["TransactionNameOverflow"])
	assert.True(t, m["TransactionNameOverflow"].(bool))
}

func TestEventQueueStats(t *testing.T) {
	es := EventQueueStats{}
	es.NumSentAdd(1)
	assert.EqualValues(t, 1, es.numSent)

	es.NumOverflowedAdd(1)
	assert.EqualValues(t, 1, es.numOverflowed)

	es.NumFailedAdd(1)
	assert.EqualValues(t, 1, es.numFailed)

	es.TotalEventsAdd(1)
	assert.EqualValues(t, 1, es.totalEvents)

	es.SetQueueLargest(10)
	assert.EqualValues(t, 10, es.queueLargest)

	original := es
	swapped := es.CopyAndReset()
	assert.Equal(t, EventQueueStats{}, es)
	assert.Equal(t, original, *swapped)
}

func TestRateCounts(t *testing.T) {
	rc := &RateCounts{}

	rc.RequestedInc()
	assert.EqualValues(t, 1, rc.Requested())

	rc.SampledInc()
	assert.EqualValues(t, 1, rc.Sampled())

	rc.LimitedInc()
	assert.EqualValues(t, 1, rc.Limited())

	rc.TracedInc()
	assert.EqualValues(t, 1, rc.Traced())

	rc.ThroughInc()
	assert.EqualValues(t, 1, rc.Through())

	original := *rc
	cp := rc.FlushRateCounts()

	assert.Equal(t, original, *cp)
	assert.Equal(t, &RateCounts{}, rc)
}

func TestRecordSpan(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()
	now := time.Now()
	_, span := tr.Start(
		context.Background(),
		"span name",
		trace.WithTimestamp(now),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPStatusCode(200),
			semconv.HTTPMethod("GET"),
			semconv.HTTPRoute("my cool route"),
		),
	)
	span.End(trace.WithTimestamp(now.Add(1 * time.Second)))
	reg := NewLegacyRegistry(false).(*registry)

	reg.RecordSpan(span.(sdktrace.ReadOnlySpan))

	m := reg.apmMetrics.CopyAndReset(60)
	assert.NotEmpty(t, m.m)
	v := m.m["ResponseTime&true&http.method:GET&http.status_code:200&sw.is_error:false&sw.transaction:my cool route&"]
	assert.NotNil(t, v, fmt.Sprintf("Map: %v", m.m))
	// one second in microseconds
	assert.Equal(t, float64(1000000), v.Sum)
	assert.Equal(t, 1, v.Count)
	assert.Equal(t, map[string]string{
		"http.method":      "GET",
		"http.status_code": "200",
		"sw.is_error":      "false",
		"sw.transaction":   "my cool route",
	},
		v.Tags)
	assert.Equal(t, responseTime, v.Name)

	h := reg.apmHistograms.histograms
	assert.NotEmpty(t, h)
	globalHisto := h[""]
	granularHisto := h["my cool route"]
	assert.NotNil(t, globalHisto)
	assert.NotNil(t, granularHisto)
	// The histo has fuzzy but deterministic values
	assert.Equal(t, 1.001472e+06, globalHisto.hist.Mean())
	assert.Equal(t, int64(1), globalHisto.hist.TotalCount())
	assert.Equal(t, 1.001472e+06, granularHisto.hist.Mean())
	assert.Equal(t, int64(1), granularHisto.hist.TotalCount())

	reg = NewLegacyRegistry(true).(*registry)
	// Now test for AO
	reg.RecordSpan(span.(sdktrace.ReadOnlySpan))

	m = reg.apmMetrics.CopyAndReset(60)
	assert.NotEmpty(t, m.m)
	k1 := "TransactionResponseTime&true&HttpMethod:GET&TransactionName:my cool route&"
	k2 := "TransactionResponseTime&true&HttpStatus:200&TransactionName:my cool route&"
	k3 := "TransactionResponseTime&true&TransactionName:my cool route&"
	for _, key := range []string{k1, k2, k3} {
		v = m.m[key]
		assert.NotNil(t, v, fmt.Sprintf("Map: %v", m.m))
		assert.Equal(t, float64(1000000), v.Sum)
		assert.Equal(t, 1, v.Count)
		assert.Equal(t, transactionResponseTime, v.Name)
	}
	assert.Equal(t,
		map[string]string{"HttpMethod": "GET", "TransactionName": "my cool route"},
		m.m[k1].Tags,
	)
	assert.Equal(t,
		map[string]string{"HttpStatus": "200", "TransactionName": "my cool route"},
		m.m[k2].Tags,
	)
	assert.Equal(t,
		map[string]string{"TransactionName": "my cool route"},
		m.m[k3].Tags,
	)

	h = reg.apmHistograms.histograms
	assert.NotEmpty(t, h)
	globalHisto = h[""]
	granularHisto = h["my cool route"]
	assert.NotNil(t, globalHisto)
	assert.NotNil(t, granularHisto)
	// The histo has fuzzy but deterministic values
	assert.Equal(t, 1.001472e+06, globalHisto.hist.Mean())
	assert.Equal(t, int64(1), globalHisto.hist.TotalCount())
	assert.Equal(t, 1.001472e+06, granularHisto.hist.Mean())
	assert.Equal(t, int64(1), granularHisto.hist.TotalCount())
}

func TestRecordSpanErrorStatus(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()
	now := time.Now()
	_, span := tr.Start(
		context.Background(),
		"span name",
		trace.WithTimestamp(now),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPStatusCode(500),
			semconv.HTTPMethod("GET"),
			semconv.HTTPRoute("my cool route"),
		),
	)
	span.SetStatus(codes.Error, "operation failed")
	span.End(trace.WithTimestamp(now.Add(1 * time.Second)))

	reg := NewLegacyRegistry(false).(*registry)
	reg.RecordSpan(span.(sdktrace.ReadOnlySpan))

	m := reg.apmMetrics.CopyAndReset(60)
	assert.NotEmpty(t, m.m)
	v := m.m["ResponseTime&true&http.method:GET&http.status_code:500&sw.is_error:true&sw.transaction:my cool route&"]
	assert.NotNil(t, v, fmt.Sprintf("Map: %v", m.m))
	// one second in microseconds
	assert.Equal(t, float64(1000000), v.Sum)
	assert.Equal(t, 1, v.Count)
	assert.Equal(t, map[string]string{
		"http.method":      "GET",
		"http.status_code": "500",
		"sw.is_error":      "true",
		"sw.transaction":   "my cool route",
	},
		v.Tags)
	assert.Equal(t, responseTime, v.Name)

	h := reg.apmHistograms.histograms
	assert.NotEmpty(t, h)
	globalHisto := h[""]
	granularHisto := h["my cool route"]
	assert.NotNil(t, globalHisto)
	assert.NotNil(t, granularHisto)
	// The histo has fuzzy but deterministic values
	assert.Equal(t, 1.001472e+06, globalHisto.hist.Mean())
	assert.Equal(t, int64(1), globalHisto.hist.TotalCount())
	assert.Equal(t, 1.001472e+06, granularHisto.hist.Mean())
	assert.Equal(t, int64(1), granularHisto.hist.TotalCount())

	// Now test for AO
	reg = NewLegacyRegistry(true).(*registry)
	reg.RecordSpan(span.(sdktrace.ReadOnlySpan))

	m = reg.apmMetrics.CopyAndReset(60)
	assert.NotEmpty(t, m.m)
	k1 := "TransactionResponseTime&true&HttpMethod:GET&TransactionName:my cool route&"
	k2 := "TransactionResponseTime&true&HttpStatus:500&TransactionName:my cool route&"
	k3 := "TransactionResponseTime&true&TransactionName:my cool route&"
	for _, key := range []string{k1, k2, k3} {
		v = m.m[key]
		assert.NotNil(t, v, fmt.Sprintf("Map: %v", m.m))
		assert.Equal(t, float64(1000000), v.Sum)
		assert.Equal(t, 1, v.Count)
		assert.Equal(t, transactionResponseTime, v.Name)
	}
	assert.Equal(t,
		map[string]string{"HttpMethod": "GET", "TransactionName": "my cool route"},
		m.m[k1].Tags,
	)
	assert.Equal(t,
		map[string]string{"HttpStatus": "500", "TransactionName": "my cool route"},
		m.m[k2].Tags,
	)
	assert.Equal(t,
		map[string]string{"TransactionName": "my cool route"},
		m.m[k3].Tags,
	)
	h = reg.apmHistograms.histograms
	assert.NotEmpty(t, h)
	globalHisto = h[""]
	granularHisto = h["my cool route"]
	assert.NotNil(t, globalHisto)
	assert.NotNil(t, granularHisto)
	// The histo has fuzzy but deterministic values
	assert.Equal(t, 1.001472e+06, globalHisto.hist.Mean())
	assert.Equal(t, int64(1), globalHisto.hist.TotalCount())
	assert.Equal(t, 1.001472e+06, granularHisto.hist.Mean())
	assert.Equal(t, int64(1), granularHisto.hist.TotalCount())

}

func TestRecordSpanOverflow(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()
	now := time.Now()
	_, span := tr.Start(
		context.Background(),
		"span name",
		trace.WithTimestamp(now),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPStatusCode(200),
			semconv.HTTPMethod("GET"),
			semconv.HTTPRoute("my cool route"),
		),
	)
	span.End(trace.WithTimestamp(now.Add(1 * time.Second)))

	_, span2 := tr.Start(
		context.Background(),
		"span name",
		trace.WithTimestamp(now),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPStatusCode(200),
			semconv.HTTPMethod("GET"),
			semconv.HTTPRoute("this should overflow"),
		),
	)
	span2.End(trace.WithTimestamp(now.Add(1 * time.Second)))

	reg := NewLegacyRegistry(false).(*registry)
	// The cap only takes affect after the following reset
	reg.SetApmMetricsCap(1)
	reg.apmMetrics.CopyAndReset(60)
	assert.Equal(t, int32(1), reg.ApmMetricsCap())

	reg.RecordSpan(span.(sdktrace.ReadOnlySpan))
	reg.RecordSpan(span2.(sdktrace.ReadOnlySpan))

	m := reg.apmMetrics.CopyAndReset(60)
	// We expect to have a record for `my cool route` and one for `other`
	assert.Equal(t, 2, len(m.m))
	v := m.m["ResponseTime&true&http.method:GET&http.status_code:200&sw.is_error:false&sw.transaction:my cool route&"]
	assert.NotNil(t, v, fmt.Sprintf("Map: %v", m.m))
	// one second in microseconds
	assert.Equal(t, float64(1000000), v.Sum)
	assert.Equal(t, 1, v.Count)
	assert.Equal(t, map[string]string{
		"http.method":      "GET",
		"http.status_code": "200",
		"sw.is_error":      "false",
		"sw.transaction":   "my cool route",
	},
		v.Tags)
	assert.Equal(t, responseTime, v.Name)

	v = m.m["ResponseTime&true&http.method:GET&http.status_code:200&sw.is_error:false&sw.transaction:other&"]
	assert.NotNil(t, v, fmt.Sprintf("Map: %v", m.m))
	// one second in microseconds
	assert.Equal(t, float64(1000000), v.Sum)
	assert.Equal(t, 1, v.Count)
	assert.Equal(t, map[string]string{
		"http.method":      "GET",
		"http.status_code": "200",
		"sw.is_error":      "false",
		"sw.transaction":   "other",
	},
		v.Tags)
	assert.Equal(t, responseTime, v.Name)

	h := reg.apmHistograms.histograms
	assert.NotEmpty(t, h)
	globalHisto := h[""]
	granularHisto := h["my cool route"]
	assert.NotNil(t, globalHisto)
	assert.NotNil(t, granularHisto)
	// The histo has fuzzy but deterministic values
	assert.Equal(t, 1.001472e+06, globalHisto.hist.Mean())
	// `other` will have increased the global histo
	assert.Equal(t, int64(2), globalHisto.hist.TotalCount())
	assert.Equal(t, 1.001472e+06, granularHisto.hist.Mean())
	assert.Equal(t, int64(1), granularHisto.hist.TotalCount())
}

func TestRecordSpanOverflowAppoptics(t *testing.T) {
	tr, teardown := testutils.TracerSetup()
	defer teardown()
	now := time.Now()
	_, span := tr.Start(
		context.Background(),
		"span name",
		trace.WithTimestamp(now),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPStatusCode(200),
			semconv.HTTPMethod("GET"),
			semconv.HTTPRoute("my cool route"),
		),
	)
	span.End(trace.WithTimestamp(now.Add(1 * time.Second)))

	_, span2 := tr.Start(
		context.Background(),
		"span name",
		trace.WithTimestamp(now),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPStatusCode(200),
			semconv.HTTPMethod("GET"),
			semconv.HTTPRoute("this should overflow"),
		),
	)
	span2.End(trace.WithTimestamp(now.Add(1 * time.Second)))

	// The cap only takes affect after the following reset
	// Appoptics-style will generate 3 metrics, so we'll set the cap to that here
	reg := NewLegacyRegistry(true).(*registry)
	reg.SetApmMetricsCap(3)
	reg.apmMetrics.CopyAndReset(60)
	assert.Equal(t, int32(3), reg.ApmMetricsCap())

	reg.RecordSpan(span.(sdktrace.ReadOnlySpan))
	reg.RecordSpan(span2.(sdktrace.ReadOnlySpan))

	m := reg.apmMetrics.CopyAndReset(60)
	// We expect to have 3 records for `my cool route` and 3 for `other`
	assert.Equal(t, 6, len(m.m))

	expectedList := []string{
		"TransactionResponseTime&true&HttpMethod:GET&TransactionName:my cool route&",
		"TransactionResponseTime&true&HttpMethod:GET&TransactionName:other&",
		"TransactionResponseTime&true&HttpStatus:200&TransactionName:my cool route&",
		"TransactionResponseTime&true&HttpStatus:200&TransactionName:other&",
		"TransactionResponseTime&true&TransactionName:my cool route&",
		"TransactionResponseTime&true&TransactionName:other&",
	}
	for _, exp := range expectedList {
		v, ok := m.m[exp]
		assert.True(t, ok)
		assert.Equal(t, float64(1000000), v.Sum)
		assert.Equal(t, 1, v.Count)
	}

	h := reg.apmHistograms.histograms
	assert.NotEmpty(t, h)
	globalHisto := h[""]
	granularHisto := h["my cool route"]
	assert.NotNil(t, globalHisto)
	assert.NotNil(t, granularHisto)
	// The histo has fuzzy but deterministic values
	assert.Equal(t, 1.001472e+06, globalHisto.hist.Mean())
	// `other` will have increased the global histo
	assert.Equal(t, int64(2), globalHisto.hist.TotalCount())
	assert.Equal(t, 1.001472e+06, granularHisto.hist.Mean())
	assert.Equal(t, int64(1), granularHisto.hist.TotalCount())
}
