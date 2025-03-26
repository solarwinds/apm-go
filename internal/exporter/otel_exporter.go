package exporter

import (
	"context"
	"os"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

func CreateAndSetupOtelExporter(ctx context.Context) (trace.SpanExporter, error) {
	setupOtelExporterEnvironment()

	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithHeaders(map[string]string{
			"Authorization": "Bearer " + config.GetApiToken(),
		}),
	)
}

func setupOtelExporterEnvironment() {
	if os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		otelCollectorAddress := config.GetOtelCollector()
		if err := os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", otelCollectorAddress); err != nil {
			log.Warningf("could not override unset OTEL_EXPORTER_OTLP_TRACES_ENDPOINT %s", err)
		} else {
			log.Infof("Setting Otel exporter to: %s", otelCollectorAddress)
		}
	}
}
