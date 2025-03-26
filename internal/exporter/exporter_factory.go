package exporter

import (
	"context"
	"os"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/reporter"
	"go.opentelemetry.io/otel/sdk/trace"
)

func NewExporter(ctx context.Context, r reporter.Reporter) (trace.SpanExporter, error) {
	if !config.GetEnabled() {
		log.Warning("SolarWinds Observability exporter is disabled.")
		return &nullExporter{}, nil
	}

	if _, ok := os.LookupEnv("USE_LEGACY_APM_EXPORTER"); ok {
		return NewAoExporter(r), nil
	}

	return CreateAndSetupOtelExporter(ctx)
}
