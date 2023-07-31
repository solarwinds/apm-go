package entryspans

import (
	"github.com/pkg/errors"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"sync"
)

var (
	state = &entrySpans{
		spans: make(map[trace.TraceID][]trace.SpanID),
	}

	NotEntrySpan = errors.New("span is not an entry span")

	nullSpanID = trace.SpanID{}
)

type entrySpans struct {
	mut sync.RWMutex

	spans map[trace.TraceID][]trace.SpanID
}

func (e *entrySpans) push(tid trace.TraceID, sid trace.SpanID) {
	e.mut.Lock()
	defer e.mut.Unlock()
	var list []trace.SpanID
	var ok bool
	if list, ok = e.spans[tid]; ok {
		list = append(list, sid)
	} else {
		list = []trace.SpanID{sid}
	}
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
			return list[0], true
		} else {
			item := list[l-1]
			list = list[:l-1]
			e.spans[tid] = list
			return item, true
		}
	} else {
		return nullSpanID, ok
	}
}

func (e *entrySpans) current(tid trace.TraceID) (trace.SpanID, bool) {
	e.mut.Lock()
	defer e.mut.Unlock()

	if list, ok := e.spans[tid]; ok {
		l := len(list)
		if len(list) == 0 {
			return nullSpanID, false
		} else {
			return list[l-1], true
		}
	} else {
		return nullSpanID, false
	}
}

func Push(span sdktrace.ReadOnlySpan) error {
	if !utils.IsEntrySpan(span) {
		return NotEntrySpan
	}

	state.push(span.SpanContext().TraceID(), span.SpanContext().SpanID())
	return nil
}

func Pop(tid trace.TraceID) (trace.SpanID, bool) {
	return state.pop(tid)
}

func Current(tid trace.TraceID) (trace.SpanID, bool) {
	return state.current(tid)
}
