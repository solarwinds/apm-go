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
	"context"
	"errors"
	"time"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
)

const (
	defaultSettingsUpdateInterval  = 30 * time.Second
	defaultSettingsTimeoutInterval = 10 * time.Second
)

type settingsUpdater struct {
	updateInterval  time.Duration
	timeoutInterval time.Duration
	oboe            Oboe
	settingsService *settingsService
}

type SettingsUpdater interface {
	Start(ctx context.Context) func()
}

func NewSettingsUpdater(o Oboe) (SettingsUpdater, error) {
	if !config.GetEnabled() {
		log.Info("SolarWinds Observability APM agent is disabled. Settings are not polled from the server.")
		return newNullSettingsUpdater(), nil
	}

	parsedServiceKey, ok := config.ParsedServiceKey()
	if !ok {
		return nil, config.ErrInvalidServiceKey
	}

	settingsUrl := config.SettingsURL()

	return &settingsUpdater{
		updateInterval:  defaultSettingsUpdateInterval,
		timeoutInterval: defaultSettingsTimeoutInterval,
		oboe:            o,
		settingsService: newSettingsService(settingsUrl, parsedServiceKey.ServiceName, "", parsedServiceKey.Token),
	}, nil
}

func (su *settingsUpdater) Start(ctx context.Context) func() {
	ctx, cancel := context.WithCancel(ctx)
	go su.run(ctx, cancel)
	return cancel
}

func (su *settingsUpdater) run(ctx context.Context, cancel context.CancelFunc) {
	defer log.Info("http settings updater exiting.")

	updateTimer := time.NewTimer(0) // Execute immediately on startup
	timeoutTimer := time.NewTimer(su.timeoutInterval)
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
		case <-ctx.Done():
			log.Infof("http settings updater requested to stop: %v", ctx.Err())
			return
		case <-updateTimer.C:
			// Only start new execution if previous one has completed
			select {
			case <-updateReady:
				go func() {
					if !su.getAndUpdateSettings(ctx, updateReady) {
						// Invalid service key - stop polling
						cancel()
					}
				}()
			default:
				// Previous execution still running, skip this tick
			}
			updateTimer.Reset(su.updateInterval)
		case <-timeoutTimer.C:
			// Only start new execution if previous one has completed
			select {
			case <-timeoutReady:
				go su.timeoutSettings(timeoutReady)
			default:
				// Previous execution still running, skip this tick
			}
			timeoutTimer.Reset(su.timeoutInterval)
		}
	}
}

func (su *settingsUpdater) getAndUpdateSettings(ctx context.Context, ready chan bool) bool {
	defer func() { ready <- true }()

	settings, err := su.getSettings(ctx)
	if err == nil {
		log.Debugf("Retrieved sampling settings: %+v", settings)
		su.oboe.UpdateSetting(settings.ToSettingsUpdateArgs())
		return true
	} else if errors.Is(err, config.ErrInvalidServiceKey) {
		log.Errorf("invalid service key, stopping settings updater: %v", err)
		return false
	} else {
		log.Warningf("failed to retrieve sampling settings: %v", err)
		return true
	}
}

func (su *settingsUpdater) getSettings(ctx context.Context) (*httpSettings, error) {
	return su.settingsService.getSettings(ctx)
}

func (su *settingsUpdater) timeoutSettings(ready chan bool) {
	defer func() { ready <- true }()

	su.oboe.CheckSettingsTimeout()
	if !su.oboe.HasDefaultSetting() {
		log.Warning("sampling settings expired, APM library may not be functioning correctly")
	}
}
