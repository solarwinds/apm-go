package instance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestInstanceDetectorReturnsValue(t *testing.T) {
	detector := instanceDetector{}

	res, err := detector.Detect(context.Background())

	require.NoError(t, err)
	require.Contains(t, res.Attributes(), attribute.KeyValue{Key: semconv.ServiceInstanceIDKey, Value: attribute.StringValue(Id)})
}
