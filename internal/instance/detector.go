package instance

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type instanceDetector struct {
}

func WithInstanceDetector() resource.Option {
	return resource.WithDetectors(instanceDetector{})
}

func (detector instanceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		{Key: semconv.ServiceInstanceIDKey, Value: attribute.StringValue(Id)},
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}
