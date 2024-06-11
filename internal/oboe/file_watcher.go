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
	"os"
	"time"

	"github.com/solarwinds/apm-go/internal/log"
)

const (
	settingsCheckDuration time.Duration = 10 * time.Second
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
		s.sType,
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
