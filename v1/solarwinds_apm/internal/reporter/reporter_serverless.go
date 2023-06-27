// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
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
	"io"

	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/config"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/metrics"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/utils"
)

type serverlessReporter struct {
	customMetrics *metrics.Measurements
	// the event writer for AWS Lambda
	logWriter FlushWriter
	// the http span
	span metrics.HTTPSpanMessage
}

func newServerlessReporter(writer io.Writer) reporter {
	r := &serverlessReporter{
		customMetrics: metrics.NewMeasurements(true, 500),
	}

	r.logWriter = newLogWriter(false, writer, 260000)

	updateSetting(int32(TYPE_DEFAULT),
		"",
		[]byte("OVERRIDE,SAMPLE_START,SAMPLE_THROUGH_ALWAYS"),
		int64(config.GetSampleRate()), 120,
		argsToMap(config.GetTokenBucketCap(),
			config.GetTokenBucketRate(),
			20.000000,
			1.000000,
			6.000000,
			0.100000,
			60,
			-1,
			[]byte("")))

	log.Warningf("The reporter (v%v, go%v) for Lambda is ready.", utils.Version(), utils.GoVersion())

	return r
}

// called when an event should be reported
func (sr *serverlessReporter) reportEvent(ctx *oboeContext, e *event) error {
	if err := prepareEvent(ctx, e); err != nil {
		// don't continue if preparation failed
		return err
	}

	_, err := sr.logWriter.Write(EventWT, (*e).bbuf.GetBuf())
	return err
}

// called when a status (e.g. __Init message) should be reported
func (sr *serverlessReporter) reportStatus(ctx *oboeContext, e *event) error {
	if err := prepareEvent(ctx, e); err != nil {
		// don't continue if preparation failed
		return err
	}

	_, err := sr.logWriter.Write(EventWT, (*e).bbuf.GetBuf())
	return err
}

// Shutdown closes the reporter.
func (sr *serverlessReporter) Shutdown(ctx context.Context) error {
	return sr.ShutdownNow()
}

// ShutdownNow closes the reporter immediately
func (sr *serverlessReporter) ShutdownNow() error {
	return nil
}

// Closed returns if the reporter is already closed.
func (sr *serverlessReporter) Closed() bool {
	return false
}

// WaitForReady waits until the reporter becomes ready or the context is canceled.
func (sr *serverlessReporter) WaitForReady(context.Context) bool {
	return true
}

func (sr *serverlessReporter) sendServerlessMetrics() {
	var messages [][]byte

	setting, ok := getSetting("")
	if !ok {
		return
	}

	inbound := metrics.BuildServerlessMessage(sr.span, FlushRateCounts(), setting.value, int(setting.source))
	if inbound != nil {
		messages = append(messages, inbound)
	}

	custom := metrics.BuildMessage(sr.customMetrics.CopyAndReset(0), true)
	if custom != nil {
		messages = append(messages, custom)
	}

	sr.sendMetrics(messages)
}

func (sr *serverlessReporter) sendMetrics(msgs [][]byte) {
	for _, msg := range msgs {
		if _, err := sr.logWriter.Write(MetricWT, msg); err != nil {
			log.Warningf("sendMetrics: %s", err)
		}
	}
}

func (sr *serverlessReporter) Flush() error {
	sr.sendServerlessMetrics()
	return sr.logWriter.Flush()
}

func (sr *serverlessReporter) SetServiceKey(string) {}

func (sr *serverlessReporter) IsAppoptics() bool {
	return false
}
