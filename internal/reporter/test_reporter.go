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
	"errors"
	"log"
	"sync"
	"time"
)

// TestReporter appends reported events to Bufs if ShouldTrace is true.
type TestReporter struct {
	EventBufs      [][]byte
	ShouldTrace    bool
	ShouldError    bool
	UseSettings    bool
	SettingType    int
	CaptureMetrics bool
	ErrorEvents    map[int]bool // whether to drop an event
	done           chan int
	wg             sync.WaitGroup
	eventChan      chan []byte
	Timeout        time.Duration
}

const (
	defaultTestReporterTimeout = 2 * time.Second
)

var oldReporter Reporter = &nullReporter{}

// TestReporterOption values may be passed to SetTestReporter.
type TestReporterOption func(*TestReporter)

func TestReporterSettingType(tp int) TestReporterOption {
	return func(r *TestReporter) { r.SettingType = tp }
}

// SetTestReporter sets and returns a test Reporter that captures raw event bytes
// for making assertions about using the graphtest package.
func SetTestReporter(options ...TestReporterOption) *TestReporter {
	r := &TestReporter{
		ShouldTrace: true,
		UseSettings: true,
		Timeout:     defaultTestReporterTimeout,
		done:        make(chan int),
		eventChan:   make(chan []byte),
	}
	for _, option := range options {
		option(r)
	}
	r.wg.Add(1)
	go r.resultWriter()

	// TODO
	// start with clean slate
	//oboe.ResetSettings()

	return r
}

func (r *TestReporter) SetServiceKey(string) error {
	return nil
}

func (r *TestReporter) GetServiceName() string {
	return "test-reporter-service"
}

func (r *TestReporter) resultWriter() {
	var numBufs int
	for {
		select {
		case numBufs = <-r.done:
			if len(r.EventBufs) >= numBufs {
				r.wg.Done()
				return
			}
			r.done = nil
		case <-time.After(r.Timeout):
			r.wg.Done()
			return
		case buf := <-r.eventChan:
			r.EventBufs = append(r.EventBufs, buf)
			if r.done == nil && len(r.EventBufs) >= numBufs {
				r.wg.Done()
				return
			}
		}
	}
}

// Close stops the test reporter from listening for events; r.EventBufs will no longer be updated and any
// calls to WritePacket() will panic.
func (r *TestReporter) Close(numBufs int) {
	r.done <- numBufs
	// wait for reader goroutine to receive numBufs events, or timeout.
	r.wg.Wait()
	close(r.eventChan)
	received := len(r.EventBufs)
	if received < numBufs {
		log.Printf("# FIX: TestReporter.Close() waited for %d events, got %d", numBufs, received)
	}
	if _, ok := oldReporter.(*nullReporter); !ok {
		oldReporter = &nullReporter{}
	}
}

// Shutdown closes the Test reporter TODO: not supported
func (r *TestReporter) Shutdown(context.Context) error {
	// return r.conn.Close()
	return errors.New("shutdown is not supported by TestReporter")
}

// ShutdownNow closes the Test reporter immediately
func (r *TestReporter) ShutdownNow() {}

// Closed returns if the reporter is closed or not TODO: not supported
func (r *TestReporter) Closed() bool {
	return false
}

// WaitForReady checks the state of the reporter and may wait for up to the specified
// duration until it becomes ready.
func (r *TestReporter) WaitForReady(context.Context) bool {
	return true
}

func (r *TestReporter) ReportEvent(Event) error {
	return errors.New("TestReporter.ReportEvent not implemented")
}

func (r *TestReporter) ReportStatus(Event) error {
	return errors.New("TestReporter.ReportStatus not implemented")
}
