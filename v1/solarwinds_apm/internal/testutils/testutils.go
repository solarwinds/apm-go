package testutils

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TracerSetup() (trace.Tracer, func()) {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(newDummyExporter()),
		sdktrace.WithSampler(newDummySampler()),
	)
	otel.SetTracerProvider(tp)
	tr := otel.Tracer(
		"foo123",
		trace.WithInstrumentationVersion("123"),
		trace.WithSchemaURL("https://www.schema.url/foo123"),
	)

	return tr, func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
		}
	}
}

type dummySampler struct{}

func (ds *dummySampler) ShouldSample(sdktrace.SamplingParameters) sdktrace.SamplingResult {
	return sdktrace.SamplingResult{
		Decision: sdktrace.RecordAndSample,
	}
}

func (ds *dummySampler) Description() string {
	return "Dummy Sampler"
}

func newDummySampler() sdktrace.Sampler {
	return &dummySampler{}
}

type dummyExporter struct{}

func newDummyExporter() *dummyExporter {
	return &dummyExporter{}
}

func (de *dummyExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (de *dummyExporter) Shutdown(context.Context) error {
	return nil
}
