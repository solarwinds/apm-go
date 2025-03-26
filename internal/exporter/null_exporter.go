package exporter

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type nullExporter struct{}

func (e *nullExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *nullExporter) Shutdown(_ context.Context) error {
	return nil
}
