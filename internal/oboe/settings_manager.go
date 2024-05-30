// © 2024 SolarWinds Worldwide, LLC. All rights reserved.
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

package oboe

import (
	"fmt"
	"time"

	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/oboe"
	"github.com/solarwinds/apm-go/internal/reporter"
)

const (
	// TODO: use time.Time
	grpcGetAndUpdateSettingsIntervalDefault = 30 // default settings retrieval interval in seconds
	grpcSettingsTimeoutCheckIntervalDefault = 10 // default check interval for timed out settings in seconds
)

type settingsManager struct {
	getAndUpdateSettingsInterval int       // settings retrieval interval in seconds
	settingsTimeoutCheckInterval int       // check interval for timed out settings in seconds
	o                            oboe.Oboe // instance of Oboe to directly UpdateSetting
	// TODO make optional
	r reporter.Reporter // instance of gRPCReporter for remote settings
	// TODO optional fileBasedWatcher
}

func NewSettingsManager(o *oboe.Oboe, r *reporter.Reporter) (*settingsManager, error) {
	// TODO make reporter optional
	if o == nil || r == nil {
		return nil, fmt.Errorf("oboe nor reporter must not be nil")
	}
	return &settingsManager{
		getAndUpdateSettingsInterval: grpcGetAndUpdateSettingsIntervalDefault,
		settingsTimeoutCheckInterval: grpcSettingsTimeoutCheckIntervalDefault,
		o:                            *o,
		r:                            *r,
	}, nil
}

// Start launches long-running goroutine periodicTasks() which
// kicks off periodic tasks to manage sample setting.
func (sm *settingsManager) Start() {
	go sm.periodicTasks()
}

// periodicTasks is a long-running goroutine to manage sample setting.
func (sm *settingsManager) periodicTasks() {
	defer log.Info("periodicTasks goroutine exiting.")

	// set up tickers
	getAndUpdateSettingsTicker := time.NewTimer(0)
	settingsTimeoutCheckTicker := time.NewTimer(time.Duration(sm.settingsTimeoutCheckInterval) * time.Second)

	defer func() {
		getAndUpdateSettingsTicker.Stop()
		settingsTimeoutCheckTicker.Stop()
	}()

	// set up 'ready' channels to indicate if a goroutine has terminated
	getAndUpdateSettingsReady := make(chan bool, 1)
	settingsTimeoutCheckReady := make(chan bool, 1)
	getAndUpdateSettingsReady <- true
	settingsTimeoutCheckReady <- true

	for {
		select {
		case <-getAndUpdateSettingsTicker.C: // get settings from collector
			// set up ticker for next round
			getAndUpdateSettingsTicker.Reset(time.Duration(sm.getAndUpdateSettingsInterval) * time.Second)
			select {
			case <-getAndUpdateSettingsReady:
				// only kick off a new goroutine if the previous one has terminated
				go sm.getAndUpdateSettings(getAndUpdateSettingsReady)
			default:
			}
		case <-settingsTimeoutCheckTicker.C: // check for timed out settings
			// set up ticker for next round
			settingsTimeoutCheckTicker.Reset(time.Duration(sm.settingsTimeoutCheckInterval) * time.Second)
			select {
			case <-settingsTimeoutCheckReady:
				// only kick off a new goroutine if the previous one has terminated
				go sm.checkSettingsTimeout(settingsTimeoutCheckReady)
			default:
			}
		}
	}
}

// retrieves settings from source
// ready	a 'ready' channel to indicate if this routine has terminated
func (sm *settingsManager) getAndUpdateSettings(ready chan bool) {
	// notify caller that this routine has terminated (defered to end of routine)
	defer func() { ready <- true }()

	// TODO
	// (A) Excise grpcReporter's call to oboe.UpdateSetting but keep LegacyRegistry updates.
	//     Also get grpcReporter to return collector settings to this Manager.
	// or
	// (B) Add new GetSettings interface to let grpcReporter keep all original behaviour

	// TODO
	// Or get settings from file instead of remote

	// TODO
	// updateOboeSettings with remote/file settings
	// instead of foo string
	sm.updateOboeSettings("foo")
}

// updates the existing settings with the newly received
// settings	new settings
func (sm *settingsManager) updateOboeSettings(foo string) {
	// TODO
	// sm.o.UpdateSetting with remote/file settings
	// instead of single foo string
	log.Info("updateOboeSettings with ", foo)
}

// delete settings that have timed out according to their TTL
// ready	a 'ready' channel to indicate if this routine has terminated
func (sm *settingsManager) checkSettingsTimeout(ready chan bool) {
	// notify caller that this routine has terminated (defered to end of routine)
	defer func() { ready <- true }()

	sm.o.CheckSettingsTimeout()
	if !sm.o.HasDefaultSetting() {
		log.Warningf("Sampling setting expired. SolarWinds Observability APM library is not working.")
	}
}
