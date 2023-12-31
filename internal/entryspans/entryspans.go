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

package entryspans

import (
	"fmt"
	"github.com/pkg/errors"
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
	if !IsEntrySpan(span) {
		return NotEntrySpan
	}

	state.push(span.SpanContext().TraceID(), span.SpanContext().SpanID())
	return nil
}

func (e *entrySpans) delete(tid trace.TraceID, sid trace.SpanID) error {
	e.mut.Lock()
	defer e.mut.Unlock()

	if list, ok := e.spans[tid]; ok {
		found := false
		for i, elem := range list {
			if elem.spanId == sid {
				list = append(list[:i], list[i+1:]...)
				found = true
				break
			}
		}
		if found {
			if len(list) == 0 {
				delete(e.spans, tid)
			} else {
				e.spans[tid] = list
			}
			return nil
		} else {
			return errors.New("could not find span id")
		}
	} else {
		return errors.New("could not find trace id")
	}
}

func Delete(span sdktrace.ReadOnlySpan) error {
	return state.delete(
		span.SpanContext().TraceID(),
		span.SpanContext().SpanID(),
	)
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
	return ""
}

func IsEntrySpan(span sdktrace.ReadOnlySpan) bool {
	parent := span.Parent()
	return !parent.IsValid() || parent.IsRemote()
}
