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
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
)

type MetricsReporter struct {
	conn                  *grpcConnection
	collectMetricInterval int32 // metrics flush interval in seconds
	registry              metrics.LegacyRegistry
	serviceKey            string
	oboe                  oboe.Oboe
	cancelled             <-chan struct{}
	done                  chan struct{}
	cancel                context.CancelFunc
	shutdownOnce          sync.Once
	wg                    sync.WaitGroup
}

func CreatePeriodicMetricsReporter(ctx context.Context, conn *grpcConnection, registry metrics.LegacyRegistry, oboe oboe.Oboe) *MetricsReporter {
	ctx, cancel := context.WithCancel(ctx)

	conn.AddClient()

	r := &MetricsReporter{
		conn:                  conn,
		collectMetricInterval: metrics.ReportingIntervalDefault,
		serviceKey:            config.GetServiceKey(),
		oboe:                  oboe,
		cancelled:             ctx.Done(),
		done:                  make(chan struct{}),
		cancel:                cancel,
		registry:              registry,
	}
	return r
}

func (mr *MetricsReporter) WithReportingInterval(interval int32) *MetricsReporter {
	atomic.StoreInt32(&mr.collectMetricInterval, interval)
	return mr
}

func (mr *MetricsReporter) Start() {
	collectMetricsTicker := time.NewTimer(mr.collectMetricsNextInterval())
	defer func() {
		collectMetricsTicker.Stop()
	}()

	// set up 'ready' channels to indicate if a goroutine has terminated
	collectMetricsReady := make(chan bool, 1)
	collectMetricsReady <- true

	mr.wg.Add(1)

	go func() {
		defer mr.wg.Done()
		for {
			select {
			case <-mr.cancelled:
				select {
				case <-collectMetricsReady:
					mr.collectMetrics(collectMetricsReady)
				default:
				}
				<-collectMetricsReady
				return
			case <-collectMetricsTicker.C: // collect and send metrics
				// set up ticker for next round
				collectMetricsTicker.Reset(mr.collectMetricsNextInterval())
				select {
				case <-collectMetricsReady:
					// only kick off a new goroutine if the previous one has terminated
					go mr.collectMetrics(collectMetricsReady)
				default:
				}
			}
		}
	}()
}

func (mr *MetricsReporter) collectMetricsNextInterval() time.Duration {
	i := int(atomic.LoadInt32(&mr.collectMetricInterval))
	interval := i - (time.Now().Second() % i)
	return time.Duration(interval) * time.Second
}

// collects the current metrics, puts them on the channel, and kicks off sendMetrics()
// collectReady	a 'ready' channel to indicate if this routine has terminated
func (mr *MetricsReporter) collectMetrics(collectReady chan bool) {
	// notify caller that this routine has terminated (defered to end of routine)
	defer func() { collectReady <- true }()

	i := atomic.LoadInt32(&mr.collectMetricInterval)

	var messages [][]byte
	// generate a new metrics message
	// colleciton of metrics
	builtin := mr.registry.BuildBuiltinMetricsMessage(i, mr.conn.queueStats.CopyAndReset(), mr.oboe.FlushRateCounts(), config.GetRuntimeMetrics())
	if builtin != nil {
		messages = append(messages, builtin)
	}

	custom := mr.registry.BuildCustomMetricsMessage(i)
	if custom != nil {
		messages = append(messages, custom)
	}

	mr.sendMetrics(messages)
}

// listens on the metrics message channel, collects all messages on that channel and
// attempts to send them to the collector using the GRPC method PostMetrics()
func (mr *MetricsReporter) sendMetrics(msgs [][]byte) {
	// no messages on the channel so nothing to send, return
	if len(msgs) == 0 {
		return
	}

	method := newPostMetricsMethod(mr.serviceKey, msgs)

	if err := mr.conn.InvokeRPC(mr.done, method); err == nil {
		log.Info(method.CallSummary())
	} else if errors.Is(err, errInvalidServiceKey) {
		mr.cancel()
	} else {
		log.Warningf("sendMetrics: %s", err)
	}
}

func (mr *MetricsReporter) Shutdown() {
	mr.shutdownOnce.Do(func() {
		mr.cancel()
		mr.wg.Wait()    // wait to flush metrics
		mr.conn.Close() // release client and close grpc connection
		close(mr.done)
	})
}
