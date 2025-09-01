// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reporter

import (
	"context"
	"testing"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/stretchr/testify/require"
)

func TestMetricsIsFlueshedOnReporterShutdown(t *testing.T) {
	ctx := context.Background()
	addr := "localhost:4567"
	t.Setenv("SW_APM_COLLECTOR", addr)
	t.Setenv("SW_APM_TRUSTEDPATH", testCertFile)
	config.Load()
	host.Start()
	server := StartTestGRPCServer(t, addr)
	time.Sleep(100 * time.Millisecond)
	grpcConn, err := CreateGrpcConnection()
	require.NoError(t, err)

	metricsReporter := CreatePeriodicMetricsReporter(ctx, grpcConn, metrics.NewLegacyRegistry(false), oboe.NewOboe()).WithReportingInterval(5)
	metricsReporter.Start()
	time.Sleep(100 * time.Millisecond)
	metricsReporter.Shutdown()
	server.Stop()

	require.NotEmpty(t, server.metrics)
}
