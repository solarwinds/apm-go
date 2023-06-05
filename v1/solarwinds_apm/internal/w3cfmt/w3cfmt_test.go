package w3cfmt

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

const spanIdHex = "0123456789abcdef"

var spanId, err = trace.SpanIDFromHex(spanIdHex)

func init() {
	if err != nil {
		log.Fatal("Fatal error: ", err)
	}
}

func TestSwFromCtx(t *testing.T) {
	sc := trace.SpanContext{}.WithSpanID(spanId).WithTraceFlags(trace.TraceFlags(0x00))

	assert.Equal(t, fmt.Sprintf("%s-00", spanIdHex), SwFromCtx(sc))

	sc = sc.WithTraceFlags(trace.TraceFlags(0x01))
	assert.Equal(t, fmt.Sprintf("%s-01", spanIdHex), SwFromCtx(sc))
}
