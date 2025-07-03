package k8s

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

type resourceDetector struct {
}

func NewResourceDetector() resource.Detector {
	return &resourceDetector{}
}

func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{}

	if metadata := MemoizeMetadata(); metadata != nil {
		attributes = append(attributes,
			semconv.K8SPodUID(metadata.PodUid),
			semconv.K8SPodName(metadata.PodName),
			semconv.K8SNamespaceName(metadata.Namespace),
		)

	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}
