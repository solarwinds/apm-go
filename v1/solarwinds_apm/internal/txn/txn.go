package txn

import (
	"go.opentelemetry.io/otel/trace"
	"sync"
)

var (
	mut  sync.RWMutex
	txns = make(map[[24]byte]string)
)

func makeKey(ctx trace.SpanContext) [24]byte {
	spanId := ctx.SpanID()
	traceId := ctx.TraceID()
	key := [24]byte{}
	copy(key[:], spanId[:])
	copy(key[8:], traceId[:])
	return key
}

func Set(ctx trace.SpanContext, name string) {
	key := makeKey(ctx)
	mut.Lock()
	defer mut.Unlock()

	txns[key] = name
}

func Get(ctx trace.SpanContext) (name string, ok bool) {
	key := makeKey(ctx)
	mut.RLock()
	defer mut.RUnlock()

	name, ok = txns[key]
	return
}

func Remove(ctx trace.SpanContext) (removed bool) {
	key := makeKey(ctx)
	mut.Lock()
	defer mut.Unlock()

	if _, ok := txns[key]; ok {
		delete(txns, key)
		return true
	}
	return false
}
