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
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/metrics"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/uams"
	"github.com/solarwinds/apm-go/internal/utils"

	"context"

	collector "github.com/solarwinds/apm-proto/go/collectorpb"
	uatomic "go.uber.org/atomic"
)

const (
	// These are hard-coded parameters for the gRPC reporter. Any of them become
	// configurable in future versions will be moved to package config.
	// TODO: use time.Time
	grpcGetAndUpdateSettingsIntervalDefault = 30               // default settings retrieval interval in seconds
	grpcSettingsTimeoutCheckIntervalDefault = 10               // default check interval for timed out settings in seconds
	grpcPingIntervalDefault                 = 20               // default interval for keep alive pings in seconds
	grpcRetryDelayInitial                   = 500              // initial connection/send retry delay in milliseconds
	grpcRetryDelayMultiplier                = 1.5              // backoff multiplier for unsuccessful retries
	grpcCtxTimeout                          = 10 * time.Second // gRPC method invocation timeout in seconds
	grpcRedirectMax                         = 20               // max allowed collector redirects
	grpcRetryLogThreshold                   = 10               // log prints after this number of retries (about 56.7s)
)

type grpcReporter struct {
	conn                         *grpcConnection // used for all RPC calls
	getAndUpdateSettingsInterval int             // settings retrieval interval in seconds
	settingsTimeoutCheckInterval int             // check interval for timed out settings in seconds

	serviceKey      *uatomic.String // service key
	otelServiceName string

	eventMessages  chan []byte // channel for event messages (sent from agent)
	statusMessages chan []byte // channel for status messages (sent from agent)

	// The reporter is considered ready if there is a valid default setting for sampling.
	// It should be accessed atomically.
	ready int32
	// The condition variable to notify those who are waiting for the reporter becomes ready
	cond *sync.Cond

	// The reporter doesn't have a explicit field to record its state. This channel is used to notify all the
	// long-running goroutines to stop themselves. All the goroutines will check this channel and close if the
	// channel is closed.
	// Don't send data into this channel, just close it by calling Shutdown().
	done       chan struct{}
	doneClosed sync.Once
	// The flag to indicate gracefully stopping the reporter. It should be accessed atomically.
	// A (default) zero value means shutdown abruptly.
	gracefully int32

	// metrics
	registry metrics.LegacyRegistry

	// oboe
	oboe oboe.Oboe
}

// gRPC reporter errors
var (
	ErrShutdownClosedReporter = errors.New("trying to shutdown a closed reporter")
	ErrShutdownTimeout        = errors.New("Shutdown timeout")
	ErrReporterIsClosed       = errors.New("the reporter is closed")
)

func getProxy() string {
	return config.GetProxy()
}

func getProxyCertPath() string {
	return config.GetProxyCertPath()
}

// initializes a new GRPC reporter from scratch (called once on program startup)
// returns	GRPC Reporter object
func newGRPCReporter(grpcConn *grpcConnection, otelServiceName string, registry metrics.LegacyRegistry, o oboe.Oboe) Reporter {
	r := &grpcReporter{
		conn: grpcConn,

		getAndUpdateSettingsInterval: grpcGetAndUpdateSettingsIntervalDefault,
		settingsTimeoutCheckInterval: grpcSettingsTimeoutCheckIntervalDefault,

		serviceKey:      uatomic.NewString(config.GetServiceKey()),
		otelServiceName: otelServiceName,

		eventMessages:  make(chan []byte, 10000),
		statusMessages: make(chan []byte, 100),

		cond: sync.NewCond(&sync.Mutex{}),
		done: make(chan struct{}),

		registry: registry,
		oboe:     o,
	}

	r.start()

	log.Warningf("The reporter (%v, v%v, go%v) is initialized. Waiting for the dynamic settings.",
		r.done, utils.Version(), utils.GoVersion())
	return r
}

func (r *grpcReporter) SetServiceKey(key string) error {
	if config.IsValidServiceKey(key) {
		r.serviceKey.Store(key)
		return nil
	}
	return errors.New("invalid service key format")
}

func (r *grpcReporter) GetServiceName() string {
	if r.otelServiceName != "" {
		return r.otelServiceName
	}
	s := strings.Split(r.serviceKey.Load(), ":")
	if len(s) != 2 {
		log.Warningf("could not extract service name from service key")
		return ""
	}
	return s[1]
}

func (r *grpcReporter) isGracefully() bool {
	return atomic.LoadInt32(&r.gracefully) == 1
}

func (r *grpcReporter) setGracefully(flag bool) {
	var i int32
	if flag {
		i = 1
	}
	atomic.StoreInt32(&r.gracefully, i)
}

func (r *grpcReporter) start() {
	// start up the host observer
	host.Start()
	uams.Start()
	// start up long-running goroutine eventSender() which listens on the events message channel
	// and reports incoming events to the collector using GRPC
	go r.eventSender()

	// start up long-running goroutine statusSender() which listens on the status message channel
	// and reports incoming events to the collector using GRPC
	go r.statusSender()

	// start up long-running goroutine periodicTasks() which kicks off periodic tasks like
	// collectMetrics() and getAndUpdateSettings()
	if !periodicTasksDisabled {
		go r.periodicTasks()
	}
}

// ShutdownNow stops the reporter immediately.
func (r *grpcReporter) ShutdownNow() {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	if err := r.Shutdown(ctx); err != nil {
		log.Warning("Received error when shutting down reporter", err)
	}
}

// Shutdown closes the reporter by close the `done` channel. All long-running goroutines
// monitor the channel `done` in the reporter and close themselves when the channel is closed.
func (r *grpcReporter) Shutdown(ctx context.Context) error {
	var err error

	select {
	case <-r.done:
		err = ErrShutdownClosedReporter
	default:
		r.doneClosed.Do(func() {
			log.Warningf("Shutting down the reporter: %v", r.done)

			var g bool
			if d, ddlSet := ctx.Deadline(); ddlSet {
				g = time.Until(d) > 0
			} else {
				g = true
			}
			r.setGracefully(g)

			close(r.done)

			r.closeConns()
			r.setReady(false)
			host.Stop()
			uams.Stop()
			log.Warning("SolarWinds Observability APM agent is stopped.")
		})
	}
	return err
}

// closeConns closes all the gRPC connections of a reporter
func (r *grpcReporter) closeConns() {
	r.conn.Close()
}

// Closed return true if the reporter is already closed, or false otherwise.
func (r *grpcReporter) Closed() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

func (r *grpcReporter) setReady(ready bool) {
	var s int32
	if ready {
		s = 1
	}
	atomic.StoreInt32(&r.ready, s)
}

func (r *grpcReporter) isReady() bool {
	return atomic.LoadInt32(&r.ready) == 1
}

// WaitForReady waits until the reporter becomes ready or the context is canceled.
//
// The reporter is still considered `not ready` if (in rare cases) the default
// setting is retrieved from the collector but expires after the TTL, and no new
// setting is fetched.
func (r *grpcReporter) WaitForReady(ctx context.Context) bool {
	if r.isReady() {
		return true
	}

	ready := make(chan struct{})
	var e int32

	go func(ch chan struct{}, exit *int32) {
		r.cond.L.Lock()
		for !r.isReady() && (atomic.LoadInt32(exit) != 1) {
			r.cond.Wait()
		}
		r.cond.L.Unlock()
		close(ch)
	}(ready, &e)

	select {
	case <-ready:
		return true
	case <-ctx.Done():
		atomic.StoreInt32(&e, 1)
		return false
	}
}

// long-running goroutine that kicks off periodic tasks like collectMetrics() and getAndUpdateSettings()
func (r *grpcReporter) periodicTasks() {
	defer log.Info("periodicTasks goroutine exiting.")

	// set up tickers
	getAndUpdateSettingsTicker := time.NewTimer(0)
	settingsTimeoutCheckTicker := time.NewTimer(time.Duration(r.settingsTimeoutCheckInterval) * time.Second)

	defer func() {
		getAndUpdateSettingsTicker.Stop()
		settingsTimeoutCheckTicker.Stop()
		r.conn.pingTicker.Stop()
	}()

	getAndUpdateSettingsReady := make(chan bool, 1)
	settingsTimeoutCheckReady := make(chan bool, 1)
	getAndUpdateSettingsReady <- true
	settingsTimeoutCheckReady <- true

	for {
		// Exit if the reporter's done channel is closed.
		select {
		case <-r.done:
			if !r.isGracefully() {
				return
			}
		default:
		}

		select {
		case <-getAndUpdateSettingsTicker.C: // get settings from collector
			// set up ticker for next round
			getAndUpdateSettingsTicker.Reset(time.Duration(r.getAndUpdateSettingsInterval) * time.Second)
			select {
			case <-getAndUpdateSettingsReady:
				// only kick off a new goroutine if the previous one has terminated
				go r.getAndUpdateSettings(getAndUpdateSettingsReady)
			default:
			}
		case <-settingsTimeoutCheckTicker.C: // check for timed out settings
			// set up ticker for next round
			settingsTimeoutCheckTicker.Reset(time.Duration(r.settingsTimeoutCheckInterval) * time.Second)
			select {
			case <-settingsTimeoutCheckReady:
				// only kick off a new goroutine if the previous one has terminated
				go r.checkSettingsTimeout(settingsTimeoutCheckReady)
			default:
			}
		case <-r.conn.pingTicker.C: // ping on event connection (keep alive)
			// set up ticker for next round
			r.conn.resetPing()
			go func() {
				if errors.Is(r.conn.ping(r.done, r.serviceKey.Load()), errInvalidServiceKey) {
					r.ShutdownNow()
				}
			}()
		}
	}
}

func (r *grpcReporter) ReportEvent(e Event) error {
	if e == nil {
		return errors.New("cannot report nil event")
	}
	select {
	case r.eventMessages <- e.ToBson():
		r.conn.queueStats.TotalEventsAdd(int64(1))
		return nil
	default:
		r.conn.queueStats.NumOverflowedAdd(int64(1))
		return errors.New("event message queue is full")
	}
}

func (r *grpcReporter) ReportStatus(e Event) error {
	if e == nil {
		return errors.New("cannot report nil event")
	}
	select {
	case r.statusMessages <- e.ToBson():
		return nil
	default:
		return errors.New("status message queue is full")
	}
}

// eventSender is a long-running goroutine that listens on the events message
// channel, collects all messages on that channel and attempts to send them to
// the collector using the gRPC method PostEvents()
func (r *grpcReporter) eventSender() {
	batches := make(chan [][]byte, 10)
	defer func() {
		close(batches)
		log.Info("eventSender goroutine exiting.")
	}()

	go r.eventBatchSender(batches)

	opts := config.ReporterOpts()
	hwm := int(opts.GetMaxReqBytes())
	if hwm <= 0 {
		log.Warningf("The event sender is disabled by setting hwm=%d", hwm)
		hwm = 0
	}

	// This event bucket is drainable either after it reaches HWM, or the flush
	// interval has passed.
	evtBucket := NewBytesBucket(r.eventMessages,
		WithHWM(hwm),
		WithGracefulShutdown(r.isGracefully()),
		WithClosingIndicator(r.done),
		WithIntervalGetter(func() time.Duration {
			return time.Second * time.Duration(opts.GetEventFlushInterval())
		}),
	)

	for {
		// Pour as much water as we can into the bucket. It blocks until it's
		// full or timeout.
		evtBucket.PourIn()
		// The events can only be pushed into the channel when the bucket
		// is drainable (either full or timeout) and we've got the token
		// to push events.
		//
		// If the token is holding by eventRetrySender, it usually means the
		// events sending is too slow (or the events are generated too fast).
		// We have to wait in this case.
		//
		// If the reporter is closing, we may have the last chance to send all
		// the queued events.
		if evtBucket.Full() {
			c := evtBucket.Count()
			dropped := evtBucket.DroppedCount()
			if dropped != 0 {
				log.Infof("Pushed %d events to the sender, dropped %d oversize events.", c, dropped)
			} else {
				log.Debugf("Pushed %d events to the sender.", c)
			}

			batches <- evtBucket.Drain()
		}

		select {
		// Check if the agent is required to quit.
		case <-r.done:
			return
		default:
		}

	}
}

func (r *grpcReporter) eventBatchSender(batches <-chan [][]byte) {
	defer func() {
		log.Info("eventBatchSender goroutine exiting.")
	}()

	var closing bool
	var messages [][]byte

	for {
		// this will block until a message arrives or the reporter is closed
		select {
		case messages = <-batches:
			if len(messages) == 0 {
				batches = nil
			}
		case <-r.done:
			select {
			case messages = <-batches:
			default:
			}
			if !r.isGracefully() {
				return
			}
			closing = true
		}

		if len(messages) != 0 {
			method := newPostEventsMethod(r.serviceKey.Load(), messages)
			if err := r.conn.InvokeRPC(r.done, method); err == nil {
				log.Info(method.CallSummary())
			} else if errors.Is(err, errInvalidServiceKey) {
				r.ShutdownNow()
			} else {
				log.Warningf("eventBatchSender: %s", err)
			}
		}

		if closing {
			return
		}
	}
}

// ================================ Settings Handling ====================================

// retrieves the settings from the collector and updates APM with them
// ready	a 'ready' channel to indicate if this routine has terminated
func (r *grpcReporter) getAndUpdateSettings(ready chan bool) {
	// notify caller that this routine has terminated (defered to end of routine)
	defer func() { ready <- true }()

	remoteSettings, err := r.getSettings()
	if err == nil {
		r.updateSettings(remoteSettings)
	} else if errors.Is(err, errInvalidServiceKey) {
		r.ShutdownNow()
	} else {
		log.Errorf("Could not getAndUpdateSettings: %s", err)
	}
}

// retrieves settings from collector and returns them
func (r *grpcReporter) getSettings() (*collector.SettingsResult, error) {
	method := newGetSettingsMethod(r.serviceKey.Load())
	if err := r.conn.InvokeRPC(r.done, method); err == nil {
		logger := log.Info
		if method.Resp.Warning != "" {
			logger = log.Warning
		}
		logger(method.CallSummary())
		return method.Resp, nil
	} else {
		log.Infof("getSettings: %s", err)
		return nil, err
	}
}

// updates the existing settings with the newly received
// settings	new settings
func (r *grpcReporter) updateSettings(settings *collector.SettingsResult) {
	for _, s := range settings.GetSettings() {
		r.oboe.UpdateSetting(s.Flags, s.Value, time.Duration(s.Ttl)*time.Second, s.Arguments)

		// update events flush interval
		o := config.ReporterOpts()
		ei := ParseInt32(s.Arguments, constants.KvEventsFlushInterval, int32(o.GetEventFlushInterval()))
		o.SetEventFlushInterval(int64(ei))

		// update MaxTransactions
		mt := ParseInt32(s.Arguments, constants.KvMaxTransactions, r.registry.ApmMetricsCap())
		r.registry.SetApmMetricsCap(mt)

		maxCustomMetrics := ParseInt32(s.Arguments, constants.KvMaxCustomMetrics, r.registry.CustomMetricsCap())
		r.registry.SetCustomMetricsCap(maxCustomMetrics)
	}

	if !r.isReady() && r.oboe.HasDefaultSetting() {
		r.cond.L.Lock()
		r.setReady(true)
		log.Warningf("Got dynamic settings. The SolarWinds Observability APM agent (%v) is ready.", r.done)
		r.cond.Broadcast()
		r.cond.L.Unlock()
	}
}

// delete settings that have timed out according to their TTL
// ready	a 'ready' channel to indicate if this routine has terminated
func (r *grpcReporter) checkSettingsTimeout(ready chan bool) {
	// notify caller that this routine has terminated (defered to end of routine)
	defer func() { ready <- true }()

	r.oboe.CheckSettingsTimeout()
	if r.isReady() && !r.oboe.HasDefaultSetting() {
		log.Warningf("Sampling setting expired. SolarWinds Observability APM library (%v) is not working.", r.done)
		r.setReady(false)
	}
}

// ========================= Status Message Handling =============================

// long-running goroutine that listens on the status message channel, collects all messages
// on that channel and attempts to send them to the collector using the GRPC method PostStatus()
func (r *grpcReporter) statusSender() {
	defer log.Info("statusSender goroutine exiting.")

	for {
		var messages [][]byte

		select {
		// this will block until a message arrives
		case e := <-r.statusMessages:
			messages = append(messages, e)
		case <-r.done: // Exit if the reporter's done channel is closed.
			return
		}
		// one message detected, see if there are more and get them all!
		done := false
		for !done {
			select {
			case e := <-r.statusMessages:
				messages = append(messages, e)
			default:
				done = true
			}
		}
		method := newPostStatusMethod(r.serviceKey.Load(), messages)
		if err := r.conn.InvokeRPC(r.done, method); err == nil {
			log.Info(method.CallSummary())
		} else if errors.Is(err, errInvalidServiceKey) {
			r.ShutdownNow()
		} else {
			log.Infof("statusSender: %s", err)
		}
	}
}
