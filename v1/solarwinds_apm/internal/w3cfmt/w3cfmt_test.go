package w3cfmt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestEncodeTraceState(t *testing.T) {
	spanIdHex := "0123456789abcdef"
	spanId, err := trace.SpanIDFromHex(spanIdHex)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("disabled: %x\n", DecisionFlagsDisabled)
	fmt.Printf("enabled: %x\n", DecisionFlagsEnabled)
	assert.Equal(t, fmt.Sprintf("%s-01", spanIdHex), encodeTraceState(spanId, DecisionFlagsEnabled))
	assert.Equal(t, fmt.Sprintf("%s-00", spanIdHex), encodeTraceState(spanId, DecisionFlagsDisabled))
}

func TestDecodeTraceParent(t *testing.T) {
	tp := "00-45883e0e30ab068640466764ee435201-4885f4c69f7aef6b-01"
	err := DecodeTraceParent(tp)
	if err != nil {
		t.Error(err)
	}
}
