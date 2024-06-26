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
	"context"
	"os"
	"time"

	"github.com/solarwinds/apm-go/internal/log"
)

const (
	settingsCheckDuration = 10 * time.Second
	settingsFileName      = "/tmp/solarwinds-apm-settings.json"

	timeoutEnv = "SW_APM_INITIAL_SETTINGS_FILE_TIMEOUT"
)

var exit = make(chan bool, 1)

type FileBasedWatcher interface {
	Start()
	Stop()
}

// NewFileBasedWatcher returns a FileBasedWatcher that periodically
// reads lambda settings from file
func NewFileBasedWatcher(oboe Oboe) FileBasedWatcher {
	return &fileBasedWatcher{
		oboe,
	}
}

type fileBasedWatcher struct {
	o Oboe
}

// readSettingFromFile parses, normalizes, and print settings from file
func (w *fileBasedWatcher) readSettingFromFile() {
	s, err := newSettingLambdaFromFile()
	if os.IsNotExist(err) {
		log.Debug("Settings file does not yet exist")
		return
	} else if err != nil {
		log.Errorf("Could not read setting from file: %s", err)
		return
	}
	log.Debugf(
		"Got lambda settings from file:\n%+v",
		s,
	)
	w.o.UpdateSetting(
		0,
		s.layer,
		s.flags,
		s.value,
		s.ttl,
		s.args,
	)
}

// Start runs a ticker that checks settings expiry from cache
// and, if expired, updates cache and oboe settings.
func (w *fileBasedWatcher) Start() {
	ticker := time.NewTicker(settingsCheckDuration)
	waitForSettingsFile()
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-exit:
				return
			case <-ticker.C:
				w.readSettingFromFile()
			}
		}
	}()
	w.readSettingFromFile()
}

func (w *fileBasedWatcher) Stop() {
	log.Info("Stopping settings file watcher.")
	exit <- true
}

func waitForSettingsFile() {
	var timeout = 1 * time.Second
	if timeoutStr := os.Getenv(timeoutEnv); timeoutStr != "" {
		if override, err := time.ParseDuration(timeoutStr); err != nil {
			log.Errorf("could not parse duration from %s '%s': %s", timeoutEnv, timeoutStr, err)
		} else if int64(override) < 1 {
			log.Infof("%s was 0 or negative, skipping wait for settings file", timeoutEnv)
			return
		} else {
			timeout = override
		}
	}
	log.Debugf("Waiting for settings file for up to %s (override with %s; set to 0 to skip)", timeout, timeoutEnv)
	// We could use something like fsnotify, but that's overkill for something this simple
	waitTicker := time.NewTicker(10 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	defer waitTicker.Stop()
	for {
		select {
		case <-waitTicker.C:
			{
				_, err := os.Stat(settingsFileName)
				if err == nil {
					log.Info("Settings file found")
					return
				} else if os.IsNotExist(err) {
					log.Debug("Settings file does not yet exist")
				} else {
					log.Errorf("Could not read settings from file: %s", err)
					return
				}
			}
		case <-ctx.Done():
			{
				log.Info("timed out waiting for settings file")
				return
			}
		}
	}
}
