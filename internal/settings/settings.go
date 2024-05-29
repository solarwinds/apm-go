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

package settings

import (
	"time"

	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/oboe"
)

const (
	// TODO: use time.Time
	grpcGetSettingsIntervalDefault          = 30 // default settings retrieval interval in seconds
	grpcSettingsTimeoutCheckIntervalDefault = 10 // default check interval for timed out settings in seconds
)

type settingsManager struct {
	getSettingsInterval          int       // settings retrieval interval in seconds
	settingsTimeoutCheckInterval int       // check interval for timed out settings in seconds
	o                            oboe.Oboe // instance of Oboe to directly UpdateSetting
}

func NewSettingsManager(oboe *oboe.Oboe) *settingsManager {
	return &settingsManager{
		getSettingsInterval:          grpcGetSettingsIntervalDefault,
		settingsTimeoutCheckInterval: grpcSettingsTimeoutCheckIntervalDefault,
		o:                            *oboe,
	}
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
	getSettingsTicker := time.NewTimer(0)
	settingsTimeoutCheckTicker := time.NewTimer(time.Duration(sm.settingsTimeoutCheckInterval) * time.Second)

	defer func() {
		getSettingsTicker.Stop()
		settingsTimeoutCheckTicker.Stop()
	}()

	// set up 'ready' channels to indicate if a goroutine has terminated
	getSettingsReady := make(chan bool, 1)
	settingsTimeoutCheckReady := make(chan bool, 1)
	getSettingsReady <- true
	settingsTimeoutCheckReady <- true

	for {
		select {
		case <-getSettingsTicker.C: // get settings from collector
			// set up ticker for next round
			getSettingsTicker.Reset(time.Duration(sm.getSettingsInterval) * time.Second)
			select {
			case <-getSettingsReady:
				// only kick off a new goroutine if the previous one has terminated
				go sm.getSettings(getSettingsReady)
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

// retrieves the settings from the collector
// ready	a 'ready' channel to indicate if this routine has terminated
func (sm *settingsManager) getSettings(ready chan bool) {
	// TODO
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
