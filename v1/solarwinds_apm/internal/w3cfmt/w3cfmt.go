package w3cfmt

import (
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

func SwFromCtx(sc trace.SpanContext) string {
	spanID := sc.SpanID()
	traceFlags := sc.TraceFlags()
	return fmt.Sprintf("%x-%x", spanID[:], []byte{byte(traceFlags)})
}
