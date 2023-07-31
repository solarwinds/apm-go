package entryspans

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"sync"
)

var (
	state = &entrySpans{
		spans: make(map[trace.TraceID][]*entrySpan),
	}

	NotEntrySpan = errors.New("span is not an entry span")

	nullSpanID    = trace.SpanID{}
	nullEntrySpan = &entrySpan{spanId: nullSpanID}
)

type entrySpan struct {
	spanId  trace.SpanID
	txnName string
}

type entrySpans struct {
	mut sync.RWMutex

	spans map[trace.TraceID][]*entrySpan
}

func (e *entrySpans) push(tid trace.TraceID, sid trace.SpanID) {
	e.mut.Lock()
	defer e.mut.Unlock()
	var list []*entrySpan
	var ok bool
	if list, ok = e.spans[tid]; !ok {
		list = []*entrySpan{}
	}
	list = append(list, &entrySpan{spanId: sid})
	e.spans[tid] = list
}

func (e *entrySpans) pop(tid trace.TraceID) (trace.SpanID, bool) {
	e.mut.Lock()
	defer e.mut.Unlock()

	if list, ok := e.spans[tid]; ok {
		l := len(list)
		if l == 0 {
			delete(e.spans, tid)
			return nullSpanID, false
		} else if l == 1 {
			delete(e.spans, tid)
			return list[0].spanId, true
		} else {
			item := list[l-1]
			list = list[:l-1]
			e.spans[tid] = list
			return item.spanId, true
		}
	} else {
		return nullSpanID, ok
	}
}

func (e *entrySpans) current(tid trace.TraceID) (*entrySpan, bool) {
	e.mut.Lock()
	defer e.mut.Unlock()
	a, ok := e.currentUnsafe(tid)
	return a, ok
}

func (e *entrySpans) currentUnsafe(tid trace.TraceID) (*entrySpan, bool) {
	if list, ok := e.spans[tid]; ok {
		l := len(list)
		if len(list) == 0 {
			return nullEntrySpan, false
		} else {
			return list[l-1], true
		}
	} else {
		return nullEntrySpan, false
	}
}

func Push(span sdktrace.ReadOnlySpan) error {
	if !utils.IsEntrySpan(span) {
		return NotEntrySpan
	}

	tid := span.SpanContext().TraceID()
	sid := span.SpanContext().SpanID()
	log.Infof("push: entry span %s %s", tid, sid)
	state.push(span.SpanContext().TraceID(), span.SpanContext().SpanID())
	return nil
}

func Pop(tid trace.TraceID) (trace.SpanID, bool) {
	sid, ok := state.pop(tid)
	if ok {
		log.Infof("pop: entry span %s %s", tid, sid)
	}
	return sid, ok
}

func Current(tid trace.TraceID) (trace.SpanID, bool) {
	curr, ok := state.current(tid)
	return curr.spanId, ok
}

func (e *entrySpans) setTransactionName(tid trace.TraceID, name string) error {
	e.mut.Lock()
	defer e.mut.Unlock()

	curr, ok := e.currentUnsafe(tid)
	if !ok {
		return fmt.Errorf("could not find entry span for trace id %s", tid)
	}
	curr.txnName = name
	return nil
}

func SetTransactionName(tid trace.TraceID, name string) error {
	return state.setTransactionName(tid, name)
}

func GetTransactionName(tid trace.TraceID) string {
	if es, ok := state.current(tid); ok {
		return es.txnName
	}
	log.Debugf("could not retrieve txn name for trace id %s", tid)
	return ""
}
