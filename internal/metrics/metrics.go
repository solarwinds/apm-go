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
	"github.com/solarwinds/apm-go/internal/bson"
	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/hdrhist"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/utils"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	metricsTransactionsMaxDefault  = 200 // default max amount of transaction names we allow per cycle
	metricsCustomMetricsMaxDefault = 500 // default max number of custom metrics
	metricsHistPrecisionDefault    = 2   // default histogram precision

	metricsTagNameLengthMax  = 64  // max number of characters for tag names
	metricsTagValueLengthMax = 255 // max number of characters for tag values

	// MaxTagsCount is the maximum number of tags allowed
	MaxTagsCount = 50

	ReportingIntervalDefault = 60 // default metrics flush interval in seconds
)

// Special transaction names
const (
	OtherTransactionName         = "other"
	MetricIDSeparator            = "&"
	TagsKVSeparator              = ":"
	otherTagExistsVal            = TagsKVSeparator + OtherTransactionName + MetricIDSeparator
	maxPathLenForTransactionName = 3
)

// Request counters definition
const (
	RequestCount               = "RequestCount"
	TraceCount                 = "TraceCount"
	TokenBucketExhaustionCount = "TokenBucketExhaustionCount"
	SampleCount                = "SampleCount"
	ThroughTraceCount          = "ThroughTraceCount"
	TriggeredTraceCount        = "TriggeredTraceCount"
)

// metric names
const (
	transactionResponseTime = "TransactionResponseTime"
	responseTime            = "ResponseTime"
)

var (
	// ErrExceedsMetricsCountLimit indicates there are too many distinct metrics.
	ErrExceedsMetricsCountLimit = errors.New("exceeds metrics count limit per flush interval")
	// ErrExceedsTagsCountLimit indicates there are too many tags
	ErrExceedsTagsCountLimit = errors.New("exceeds tags count limit")
	// ErrMetricsWithNonPositiveCount indicates the count is negative or zero
	ErrMetricsWithNonPositiveCount = errors.New("metrics with non-positive count")
)

// SpanMessage defines a span message
type SpanMessage interface {
	Process(m *measurements)
}

// BaseSpanMessage is the base span message with properties found in all types of span messages
type BaseSpanMessage struct {
	Duration time.Duration // duration of the span (nanoseconds)
	HasError bool          // boolean flag whether this transaction contains an error or not
}

// HTTPSpanMessage is used for inbound metrics
type HTTPSpanMessage struct {
	BaseSpanMessage
	Transaction string // transaction name (e.g. controller.action)
	Path        string // the url path which will be processed and used as transaction (if Transaction is empty)
	Status      int    // HTTP status code (e.g. 200, 500, ...)
	Host        string // HTTP-Host // This could be removed (-jared)
	Method      string // HTTP method (e.g. GET, POST, ...)
}

// measurement is a single measurement for reporting
type measurement struct {
	Name      string            // the name of the measurement (e.g. TransactionResponseTime)
	Tags      map[string]string // map of KVs. It may be nil
	Count     int               // count of this measurement
	Sum       float64           // sum for this measurement
	ReportSum bool              // include the sum in the report?
}

// measurements are a collection of mutex-protected measurements
type measurements struct {
	m             map[string]*measurement
	txnMap        *txnMap
	isCustom      bool
	flushInterval int32
	sync.Mutex    // protect access to this collection
}

func newMeasurements(isCustom bool, maxCount int32) *measurements {
	return &measurements{
		m:             make(map[string]*measurement),
		txnMap:        newTxnMap(maxCount),
		isCustom:      isCustom,
		flushInterval: ReportingIntervalDefault,
	}
}

func getPrecision() int {
	if precision := config.GetPrecision(); precision >= 0 && precision <= 5 {
		return precision
	} else {
		log.Errorf("value of config.Precision or SW_APM_HISTOGRAM_PRECISION must be between 0 and 5: %v", precision)
		return metricsHistPrecisionDefault
	}
}

// a single histogram
type histogram struct {
	hist *hdrhist.Hist     // internal representation of a histogram (see hdrhist package)
	tags map[string]string // map of KVs
}

// a collection of histograms
type histograms struct {
	histograms map[string]*histogram
	precision  int        // histogram precision (a value between 0-5)
	lock       sync.Mutex // protect access to this collection
}

// EventQueueStats is the counters of the event queue stats
// All the fields are supposed to be accessed through atomic operations
type EventQueueStats struct {
	numSent       int64 // number of messages that were successfully sent
	numOverflowed int64 // number of messages that overflowed the queue
	numFailed     int64 // number of messages that failed to send
	totalEvents   int64 // number of messages queued to send
	queueLargest  int64 // maximum number of messages that were in the queue at one time
}

func (s *EventQueueStats) NumSentAdd(n int64) {
	atomic.AddInt64(&s.numSent, n)
}

func (s *EventQueueStats) NumOverflowedAdd(n int64) {
	atomic.AddInt64(&s.numOverflowed, n)
}

func (s *EventQueueStats) NumFailedAdd(n int64) {
	atomic.AddInt64(&s.numFailed, n)
}

func (s *EventQueueStats) TotalEventsAdd(n int64) {
	atomic.AddInt64(&s.totalEvents, n)
}

// RateCounts is the rate counts reported by trace sampler
type RateCounts struct{ requested, sampled, limited, traced, through int64 }
type RateCountSummary struct {
	Requested, Traced, Limited, TtTraced, Sampled, Through int64
}

// FlushRateCounts reset the counters and returns the current value
func (c *RateCounts) FlushRateCounts() *RateCounts {
	return &RateCounts{
		requested: atomic.SwapInt64(&c.requested, 0),
		sampled:   atomic.SwapInt64(&c.sampled, 0),
		limited:   atomic.SwapInt64(&c.limited, 0),
		traced:    atomic.SwapInt64(&c.traced, 0),
		through:   atomic.SwapInt64(&c.through, 0),
	}
}

func (c *RateCounts) RequestedInc() {
	atomic.AddInt64(&c.requested, 1)
}

func (c *RateCounts) Requested() int64 {
	return atomic.LoadInt64(&c.requested)
}

func (c *RateCounts) SampledInc() {
	atomic.AddInt64(&c.sampled, 1)
}

func (c *RateCounts) Sampled() int64 {
	return atomic.LoadInt64(&c.sampled)
}

func (c *RateCounts) LimitedInc() {
	atomic.AddInt64(&c.limited, 1)
}

func (c *RateCounts) Limited() int64 {
	return atomic.LoadInt64(&c.limited)
}

func (c *RateCounts) TracedInc() {
	atomic.AddInt64(&c.traced, 1)
}

func (c *RateCounts) Traced() int64 {
	return atomic.LoadInt64(&c.traced)
}

func (c *RateCounts) ThroughInc() {
	atomic.AddInt64(&c.through, 1)
}

func (c *RateCounts) Through() int64 {
	return atomic.LoadInt64(&c.through)
}

// addRequestCounters add various request-related counters to the metrics message buffer.
func addRequestCounters(bbuf *bson.Buffer, index *int, rcs *RateCountSummary) {
	if rcs == nil {
		return
	}
	addMetricsValue(bbuf, index, RequestCount, rcs.Requested)
	addMetricsValue(bbuf, index, TraceCount, rcs.Traced)
	addMetricsValue(bbuf, index, TokenBucketExhaustionCount, rcs.Limited)
	addMetricsValue(bbuf, index, SampleCount, rcs.Sampled)
	addMetricsValue(bbuf, index, ThroughTraceCount, rcs.Through)
	addMetricsValue(bbuf, index, TriggeredTraceCount, rcs.TtTraced)
}

// SetCap sets the maximum number of distinct metrics allowed.
func (m *measurements) SetCap(cap int32) {
	m.txnMap.SetCap(cap)
}

// Cap returns the maximum number of distinct metrics allowed.
func (m *measurements) Cap() int32 {
	return m.txnMap.cap()
}

// CopyAndReset resets the custom metrics and return a copy of the old one.
func (m *measurements) CopyAndReset(flushInterval int32) *measurements {
	m.Lock()
	defer m.Unlock()

	clone := m.Clone()
	m.m = make(map[string]*measurement)
	m.txnMap.reset()
	m.flushInterval = flushInterval
	return clone
}

// Clone returns a shallow copy
func (m *measurements) Clone() *measurements {
	return &measurements{
		m:             m.m,
		txnMap:        m.txnMap.clone(),
		isCustom:      m.isCustom,
		flushInterval: m.flushInterval,
	}
}

// Summary submits the summary measurement to the reporter.
func (m *measurements) Summary(name string, value float64, opts MetricOptions) error {
	if err := opts.validate(); err != nil {
		return err
	}
	return m.recordWithSoloTags(name, opts.Tags, value, opts.Count, true)
}

// Increment submits the incremental measurement to the reporter.
func (m *measurements) Increment(name string, opts MetricOptions) error {
	if err := opts.validate(); err != nil {
		return err
	}
	return m.recordWithSoloTags(name, opts.Tags, 0, opts.Count, false)
}

// MetricOptions is a struct for the optional parameters of a measurement.
type MetricOptions struct {
	Count   int
	HostTag bool
	Tags    map[string]string
}

func (mo *MetricOptions) validate() error {
	if len(mo.Tags) > MaxTagsCount {
		return ErrExceedsTagsCountLimit
	}

	if mo.Count <= 0 {
		return ErrMetricsWithNonPositiveCount
	}

	return nil
}

func addRuntimeMetrics(bbuf *bson.Buffer, index *int) {
	// category runtime
	addMetricsValue(bbuf, index, "trace.go.runtime.NumGoroutine", runtime.NumGoroutine())
	addMetricsValue(bbuf, index, "trace.go.runtime.NumCgoCall", runtime.NumCgoCall())

	var mem runtime.MemStats
	host.Mem(&mem)
	// category gc
	addMetricsValue(bbuf, index, "trace.go.gc.LastGC", int64(mem.LastGC))
	addMetricsValue(bbuf, index, "trace.go.gc.NextGC", int64(mem.NextGC))
	addMetricsValue(bbuf, index, "trace.go.gc.PauseTotalNs", int64(mem.PauseTotalNs))
	addMetricsValue(bbuf, index, "trace.go.gc.NumGC", int64(mem.NumGC))
	addMetricsValue(bbuf, index, "trace.go.gc.NumForcedGC", int64(mem.NumForcedGC))
	addMetricsValue(bbuf, index, "trace.go.gc.GCCPUFraction", mem.GCCPUFraction)

	// category memory
	addMetricsValue(bbuf, index, "trace.go.memory.Alloc", int64(mem.Alloc))
	addMetricsValue(bbuf, index, "trace.go.memory.TotalAlloc", int64(mem.TotalAlloc))
	addMetricsValue(bbuf, index, "trace.go.memory.Sys", int64(mem.Sys))
	addMetricsValue(bbuf, index, "trace.go.memory.Lookups", int64(mem.Lookups))
	addMetricsValue(bbuf, index, "trace.go.memory.Mallocs", int64(mem.Mallocs))
	addMetricsValue(bbuf, index, "trace.go.memory.Frees", int64(mem.Frees))
	addMetricsValue(bbuf, index, "trace.go.memory.HeapAlloc", int64(mem.HeapAlloc))
	addMetricsValue(bbuf, index, "trace.go.memory.HeapSys", int64(mem.HeapSys))
	addMetricsValue(bbuf, index, "trace.go.memory.HeapIdle", int64(mem.HeapIdle))
	addMetricsValue(bbuf, index, "trace.go.memory.HeapInuse", int64(mem.HeapInuse))
	addMetricsValue(bbuf, index, "trace.go.memory.HeapReleased", int64(mem.HeapReleased))
	addMetricsValue(bbuf, index, "trace.go.memory.HeapObjects", int64(mem.HeapObjects))
	addMetricsValue(bbuf, index, "trace.go.memory.StackInuse", int64(mem.StackInuse))
	addMetricsValue(bbuf, index, "trace.go.memory.StackSys", int64(mem.StackSys))
}

// append host ID to a BSON buffer
// bbuf	the BSON buffer to append the KVs to
func appendHostId(bbuf *bson.Buffer) {
	if host.ConfiguredHostname() != "" {
		bbuf.AppendString("ConfiguredHostname", host.ConfiguredHostname())
	}
	appendUname(bbuf)
	bbuf.AppendString("Distro", host.Distro())
	appendIPAddresses(bbuf)
}

// gets and appends IP addresses to a BSON buffer
// bbuf	the BSON buffer to append the KVs to
func appendIPAddresses(bbuf *bson.Buffer) {
	addrs := host.IPAddresses()
	if addrs == nil {
		return
	}

	start := bbuf.AppendStartArray("IPAddresses")
	for i, address := range addrs {
		bbuf.AppendString(strconv.Itoa(i), address)
	}
	bbuf.AppendFinishObject(start)
}

// gets and appends MAC addresses to a BSON buffer
// bbuf	the BSON buffer to append the KVs to
func appendMACAddresses(bbuf *bson.Buffer, macs []string) {
	start := bbuf.AppendStartArray("MACAddresses")
	for _, mac := range macs {
		if mac == "" {
			continue
		}
		i := 0
		bbuf.AppendString(strconv.Itoa(i), mac)
		i++
	}
	bbuf.AppendFinishObject(start)
}

// appends a metric to a BSON buffer, the form will be:
//
//	{
//	  "name":"myName",
//	  "value":0
//	}
//
// bbuf		the BSON buffer to append the metric to
// index	a running integer (0,1,2,...) which is needed for BSON arrays
// name		key name
// value	value (type: int, int64, float32, float64)
func addMetricsValue(bbuf *bson.Buffer, index *int, name string, value interface{}) {
	start := bbuf.AppendStartObject(strconv.Itoa(*index))
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%v", err)
		}
	}()

	bbuf.AppendString("name", name)
	switch v := value.(type) {
	case int:
		bbuf.AppendInt("value", v)
	case int64:
		bbuf.AppendInt64("value", v)
	case float32:
		v32 := v
		v64 := float64(v32)
		bbuf.AppendFloat64("value", v64)
	case float64:
		bbuf.AppendFloat64("value", v)
	default:
		bbuf.AppendString("value", "unknown")
	}

	bbuf.AppendFinishObject(start)
	*index += 1
}

// GetTransactionFromPath performs fingerprinting on a given escaped path to extract the transaction name
// We can get the path so there is no need to parse the full URL.
// e.g. Escaped Path path: /solarwinds/apm-go/blob/metrics becomes /solarwinds/apm-go
func GetTransactionFromPath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	p := strings.Split(path, "/")
	lp := len(p)
	if lp > maxPathLenForTransactionName {
		lp = maxPathLenForTransactionName
	}
	return strings.Join(p[0:lp], "/")
}

func (s *HTTPSpanMessage) appOpticsTagsList() []map[string]string {
	var tagsList []map[string]string

	// primary key: TransactionName
	primaryTags := make(map[string]string)
	primaryTags["TransactionName"] = s.Transaction
	tagsList = append(tagsList, primaryTags)

	// secondary keys: HttpMethod, HttpStatus, Errors
	withMethodTags := utils.CopyMap(&primaryTags)
	withMethodTags["HttpMethod"] = s.Method
	tagsList = append(tagsList, withMethodTags)

	withStatusTags := utils.CopyMap(&primaryTags)
	withStatusTags["HttpStatus"] = strconv.Itoa(s.Status)
	tagsList = append(tagsList, withStatusTags)

	if s.HasError {
		withErrorTags := utils.CopyMap(&primaryTags)
		withErrorTags["Errors"] = "true"
		tagsList = append(tagsList, withErrorTags)
	}

	return tagsList
}

// processes HTTP measurements, record one for primary key, and one for each secondary key
// transactionName	the transaction name to be used for these measurements
func (s *HTTPSpanMessage) processMeasurements(metricName string, tagsList []map[string]string,
	m *measurements) error {
	if tagsList == nil {
		return errors.New("tagsList must not be nil")
	}
	duration := float64(s.Duration / time.Microsecond)
	return m.record(metricName, tagsList, duration, 1, true)
}

func (m *measurements) recordWithSoloTags(name string, tags map[string]string,
	value float64, count int, reportValue bool) error {
	return m.record(name, []map[string]string{tags}, value, count, reportValue)
}

// records a measurement
// name			key name
// tagsList		the list of the additional tags
// value		measurement value
// count		measurement count
// reportValue	should the sum of all values be reported?
func (m *measurements) record(name string, tagsList []map[string]string,
	value float64, count int, reportValue bool) error {
	if len(tagsList) == 0 {
		return nil
	}

	idTagsMap := make(map[string]map[string]string)
	idPrefixList := []string{name, strconv.FormatBool(reportValue)}

	for _, tags := range tagsList {
		idList := append(idPrefixList[:0:0], idPrefixList...)
		if tags != nil {
			// tags are part of the ID but since there's no guarantee that the map items
			// are always iterated in the same order, we need to sort them ourselves
			var tagsSorted []string
			for k, v := range tags {
				tagsSorted = append(tagsSorted, k+TagsKVSeparator+v)
			}
			sort.Strings(tagsSorted)

			idList = append(idList, tagsSorted...)
		}
		idList = append(idList, "")
		id := strings.Join(idList, MetricIDSeparator)

		idTagsMap[id] = tags
	}

	var me *measurement
	var ok bool

	// create a new measurement if it doesn't exist
	// the lock protects both measurements and measurement
	m.Lock()
	defer m.Unlock()
	for id, tags := range idTagsMap {
		if me, ok = m.m[id]; !ok {
			// N.B. This overflow logic is a bit cumbersome and is ripe for a rewrite
			if strings.Contains(id, otherTagExistsVal) ||
				m.txnMap.isWithinLimit(id) {
				me = &measurement{
					Name:      name,
					Tags:      tags,
					ReportSum: reportValue,
				}
				m.m[id] = me
			} else {
				return ErrExceedsMetricsCountLimit
			}
		}

		// add count and value
		me.Count += count
		me.Sum += value
	}
	return nil
}

// records a histogram
// hi		collection of histograms that this histogram should be added to
// name		key name
// duration	span duration
func (hi *histograms) recordHistogram(name string, duration time.Duration) {
	hi.lock.Lock()
	defer func() {
		hi.lock.Unlock()
		if err := recover(); err != nil {
			log.Errorf("Failed to record histogram: %v", err)
		}
	}()

	histograms := hi.histograms
	id := name

	tags := make(map[string]string)
	if name != "" {
		tags["TransactionName"] = name
	}

	var h *histogram
	var ok bool

	// create a new histogram if it doesn't exist
	if h, ok = histograms[id]; !ok {
		h = &histogram{
			hist: hdrhist.WithConfig(hdrhist.Config{
				LowestDiscernible: 1,
				HighestTrackable:  3600000000,
				SigFigs:           int32(hi.precision),
			}),
			tags: tags,
		}
		histograms[id] = h
	}

	// record histogram
	h.hist.Record(int64(duration / time.Microsecond))
}

// adds a measurement to a BSON buffer
// bbuf		the BSON buffer to append the metric to
// index	a running integer (0,1,2,...) which is needed for BSON arrays
// m		measurement to be added
func addMeasurementToBSON(bbuf *bson.Buffer, index *int, m *measurement) {
	start := bbuf.AppendStartObject(strconv.Itoa(*index))

	bbuf.AppendString("name", m.Name)
	bbuf.AppendInt("count", m.Count)
	if m.ReportSum {
		bbuf.AppendFloat64("sum", m.Sum)
	}

	if len(m.Tags) > 0 {
		start := bbuf.AppendStartObject("tags")
		for k, v := range m.Tags {
			if len(k) > metricsTagNameLengthMax {
				k = k[0:metricsTagNameLengthMax]
			}
			if len(v) > metricsTagValueLengthMax {
				v = v[0:metricsTagValueLengthMax]
			}
			bbuf.AppendString(k, v)
		}
		bbuf.AppendFinishObject(start)
	}

	bbuf.AppendFinishObject(start)
	*index += 1
}

// adds a histogram to a BSON buffer
// bbuf		the BSON buffer to append the metric to
// index	a running integer (0,1,2,...) which is needed for BSON arrays
// h		histogram to be added
func addHistogramToBSON(bbuf *bson.Buffer, index *int, h *histogram) {
	// get 64-base encoded representation of the histogram
	data, err := hdrhist.EncodeCompressed(h.hist)
	if err != nil {
		log.Errorf("Failed to encode histogram: %v", err)
		return
	}

	start := bbuf.AppendStartObject(strconv.Itoa(*index))

	bbuf.AppendString("name", transactionResponseTime)
	bbuf.AppendString("value", string(data))

	// append tags
	if len(h.tags) > 0 {
		start := bbuf.AppendStartObject("tags")
		for k, v := range h.tags {
			if len(k) > metricsTagNameLengthMax {
				k = k[0:metricsTagNameLengthMax]
			}
			if len(v) > metricsTagValueLengthMax {
				v = v[0:metricsTagValueLengthMax]
			}
			bbuf.AppendString(k, v)
		}
		bbuf.AppendFinishObject(start)
	}

	bbuf.AppendFinishObject(start)
	*index += 1
}

func (s *EventQueueStats) SetQueueLargest(count int64) {
	newVal := count

	for {
		currVal := atomic.LoadInt64(&s.queueLargest)
		if newVal <= currVal {
			return
		}
		if atomic.CompareAndSwapInt64(&s.queueLargest, currVal, newVal) {
			return
		}
	}
}

// CopyAndReset returns a copy of its current values and reset itself.
func (s *EventQueueStats) CopyAndReset() *EventQueueStats {
	c := &EventQueueStats{}

	c.numSent = atomic.SwapInt64(&s.numSent, 0)
	c.numFailed = atomic.SwapInt64(&s.numFailed, 0)
	c.totalEvents = atomic.SwapInt64(&s.totalEvents, 0)
	c.numOverflowed = atomic.SwapInt64(&s.numOverflowed, 0)
	c.queueLargest = atomic.SwapInt64(&s.queueLargest, 0)

	return c
}
