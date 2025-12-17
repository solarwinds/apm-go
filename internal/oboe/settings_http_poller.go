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

package oboe

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
)

const (
	defaultSettingsUpdateInterval  = 30 * time.Second
	defaultSettingsTimeoutInterval = 10 * time.Second
)

type settingsPoller struct {
	updateInterval  time.Duration
	timeoutInterval time.Duration
	done            chan struct{}
	shutdownOnce    sync.Once
	oboe            Oboe
	settingsService *settingsService
}

type SettingsPoller interface {
	Start()
	Shutdown()
}

func NewSettingsPoller(o Oboe) (SettingsPoller, error) {
	if !config.GetEnabled() {
		log.Info("SolarWinds Observability APM agent is disabled. Settings are not polled from the server.")
		return newNullSettingsPoller(), nil
	}

	parsedServiceKey, ok := config.ParsedServiceKey()
	if !ok {
		return nil, config.ErrInvalidServiceKey
	}

	return &settingsPoller{
		updateInterval:  defaultSettingsUpdateInterval,
		timeoutInterval: defaultSettingsTimeoutInterval,
		done:            make(chan struct{}),
		oboe:            o,
		settingsService: newSettingsService(getBaseURL(), parsedServiceKey.ServiceName, "", parsedServiceKey.Token),
	}, nil
}

func getBaseURL() string {
	collector := config.GetCollector()
	host := collector
	if idx := strings.LastIndex(collector, ":"); idx != -1 {
		host = collector[:idx]
	}
	return fmt.Sprintf("https://%s", host)
}

func (sp *settingsPoller) Start() {
	go sp.run()
}

func (sp *settingsPoller) run() {
	defer log.Info("http settings poller goroutine exiting.")

	updateTimer := time.NewTimer(0) // Execute immediately on startup
	timeoutTimer := time.NewTimer(sp.timeoutInterval)
	defer func() {
		updateTimer.Stop()
		timeoutTimer.Stop()
	}()

	// Semaphores to prevent overlapping executions
	updateReady := make(chan bool, 1)
	timeoutReady := make(chan bool, 1)
	updateReady <- true
	timeoutReady <- true

	for {
		select {
		case <-sp.done:
			return
		case <-updateTimer.C:
			// Only start new execution if previous one has completed
			select {
			case <-updateReady:
				go sp.getAndUpdateSettings(updateReady)
			default:
				// Previous execution still running, skip this tick
			}
			updateTimer.Reset(sp.updateInterval)
		case <-timeoutTimer.C:
			// Only start new execution if previous one has completed
			select {
			case <-timeoutReady:
				go sp.timeoutSettings(timeoutReady)
			default:
				// Previous execution still running, skip this tick
			}
			timeoutTimer.Reset(sp.timeoutInterval)
		}
	}
}

func (sp *settingsPoller) getAndUpdateSettings(ready chan bool) {
	defer func() { ready <- true }()

	settings, err := sp.getSettings()
	if err == nil {
		log.Debugf("Retrieved sampling settings: %+v", settings)
		sp.oboe.UpdateSetting(settings.ToSettingsUpdateArgs())
	} else if errors.Is(err, config.ErrInvalidServiceKey) {
		log.Errorf("invalid service key, shutting down sampling settings poller: %v", err)
		sp.Shutdown()
	} else {
		log.Warningf("failed to retrieve sampling settings: %v", err)
	}
}

func (sp *settingsPoller) getSettings() (*httpSettings, error) {
	return sp.settingsService.getSettings()
}

func (sp *settingsPoller) timeoutSettings(ready chan bool) {
	defer func() { ready <- true }()

	sp.oboe.CheckSettingsTimeout()
	if !sp.oboe.HasDefaultSetting() {
		log.Warning("sampling settings expired, APM library may not be functioning correctly")
	}
}

// Shutdown stops the settings poller gracefully.
func (sp *settingsPoller) Shutdown() {
	sp.shutdownOnce.Do(func() {
		close(sp.done)
	})
}
