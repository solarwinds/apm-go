// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package reporter

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/bson"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics"
)

const (
	udpAddrDefault = "127.0.0.1:7831"
)

type udpReporter struct {
	conn *net.UDPConn
}

func udpNewReporter() reporter {
	var conn *net.UDPConn

	// collector address override
	udpAddress := config.GetCollectorUDP()
	if udpAddress == "" {
		udpAddress = udpAddrDefault
	}

	serverAddr, err := net.ResolveUDPAddr("udp4", udpAddress)
	if err == nil {
		conn, err = net.DialUDP("udp4", nil, serverAddr)
	}
	if err != nil {
		log.Errorf("SolarWinds Observability failed to initialize UDP reporter: %v", err)
		return &nullReporter{}
	}

	// add default setting
	updateSetting(int32(TYPE_DEFAULT), "",
		[]byte("SAMPLE_START,SAMPLE_THROUGH_ALWAYS"),
		1000000, 120, argsToMap(16, 8, 16, 8, 16, 8, -1, -1, []byte("")))

	return &udpReporter{conn: conn}
}

func (r *udpReporter) report(ctx *oboeContext, e *event) error {
	if err := prepareEvent(ctx, e); err != nil {
		// don't continue if preparation failed
		return err
	}

	_, err := r.conn.Write((*e).bbuf.GetBuf())
	return err
}

// Shutdown closes the UDP reporter TODO: not supported
func (r *udpReporter) Shutdown(ctx context.Context) error {
	// return r.conn.Close()
	return errors.New("not implemented")
}

// ShutdownNow closes the reporter immediately.
func (r *udpReporter) ShutdownNow() error { return nil }

// Closed returns if the reporter is closed or not TODO: not supported
func (r *udpReporter) Closed() bool {
	return false
}

// WaitForReady waits until the reporter becomes ready or the context is canceled.
func (r *udpReporter) WaitForReady(ctx context.Context) bool { return true }

func (r *udpReporter) reportEvent(ctx *oboeContext, e *event) error {
	return r.report(ctx, e)
}

func (r *udpReporter) reportStatus(ctx *oboeContext, e *event) error {
	return r.report(ctx, e)
}

func (r *udpReporter) reportSpan(span metrics.SpanMessage) error {
	s := span.(*metrics.HTTPSpanMessage)
	bbuf := bson.NewBuffer()
	bbuf.AppendString("transaction", s.Transaction)
	bbuf.AppendString("url", s.Path)
	bbuf.AppendInt("status", s.Status)
	bbuf.AppendString("method", s.Method)
	bbuf.AppendBool("hasError", s.HasError)
	bbuf.AppendInt64("duration", s.Duration.Nanoseconds())
	bbuf.Finish()
	_, err := r.conn.Write(bbuf.GetBuf())
	return err
}

func (r *udpReporter) CustomSummaryMetric(name string, value float64, opts metrics.MetricOptions) error {
	return nil
}

func (r *udpReporter) CustomIncrementMetric(name string, opts metrics.MetricOptions) error {
	return nil
}

func (r *udpReporter) Flush() error         { return nil }
func (r *udpReporter) SetServiceKey(string) {}
