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
	stdlog "log"
	"time"
)

const (
	tick     = 10
	tickUnit = time.Second
)

var exit = make(chan bool, 1)

type FileBasedWatcher interface {
	UpdateSettingFromFile()
	Start()
	Stop()
}

// NewFileBasedWatcher returns a FileBasedWatcher that periodically
// does oboe.UpdateSetting using values from a settings JSON file.
func NewFileBasedWatcher(oboe *Oboe) FileBasedWatcher {
	return &fileBasedWatcher{
		*oboe,
	}
}

type fileBasedWatcher struct {
	o Oboe
}

func (fbw *fileBasedWatcher) UpdateSettingFromFile() {
	settingLambda, err := newSettingLambdaFromFile()
	if err != nil {
		stdlog.Fatalf("Could not update setting from file: %s", err)
		return
	}
	fbw.o.UpdateSetting(
		settingLambda.sType,
		settingLambda.layer,
		settingLambda.flags,
		settingLambda.value,
		settingLambda.ttl,
		settingLambda.args,
	)
}

func (fbw *fileBasedWatcher) Start() {
	ticker := time.NewTicker(tick * tickUnit)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-exit:
				return
			case <-ticker.C:
				stdlog.Print("Updating settings from file.")
				fbw.UpdateSettingFromFile()
			}
		}
	}()
}

func (fbw *fileBasedWatcher) Stop() {
	stdlog.Print("Stopping settings file watcher.")
	exit <- true
}
