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
	"encoding/binary"
	stdlog "log"
	"time"

	"github.com/coocood/freecache"
)

const (
	cacheSize            = 5 * 1024 * 1024 // 5 MB
	settingsCheckSeconds = 10
)

var keyTtl []byte = []byte("ttl")
var exit = make(chan bool, 1)

type FileBasedWatcher interface {
	Start()
	Stop()
}

// NewFileBasedWatcher returns a FileBasedWatcher that periodically
// does oboe.UpdateSetting using values from a settings JSON file,
// if cached settings have not yet expired.
func NewFileBasedWatcher(oboe *Oboe) FileBasedWatcher {
	return &fileBasedWatcher{
		*oboe,
		*freecache.NewCache(cacheSize),
	}
}

type fileBasedWatcher struct {
	o             Oboe
	settingsCache freecache.Cache
}

// updateCacheFromFile sets "ttl" as byte representation of ttl in seconds.
func (fbw *fileBasedWatcher) updateCacheFromFile(sl *settingLambdaNormalized) {
	ttlBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(ttlBytes, uint64(sl.ttl))
	fbw.settingsCache.Set(keyTtl, ttlBytes, int(sl.ttl))
}

// updateSettingFromFile updates oboe settings using normalized settings from file.
func (fbw *fileBasedWatcher) updateSettingFromFile(sl *settingLambdaNormalized) {
	fbw.o.UpdateSetting(
		sl.sType,
		sl.layer,
		sl.flags,
		sl.value,
		sl.ttl,
		sl.args,
	)
}

func (fbw *fileBasedWatcher) updateSettingAndCacheFromFile() {
	settingLambda, err := newSettingLambdaFromFile()
	if err != nil {
		stdlog.Fatalf("Could not update setting from file: %s", err)
		return
	}
	fbw.updateSettingFromFile(settingLambda)
	fbw.updateCacheFromFile(settingLambda)
}

// Start runs a ticker that checks settings expiry from cache
// and, if expired, updates cache and oboe settings.
func (fbw *fileBasedWatcher) Start() {
	ticker := time.NewTicker(settingsCheckSeconds * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-exit:
				return
			case <-ticker.C:
				_, expireAt, err := fbw.settingsCache.GetWithExpiration(keyTtl)
				if err != nil {
					stdlog.Fatalf("There was an issue with settingsCache: %s", err)
					// TODO: disable APM Go
				} else {
					// If cached settings expired, update cache and
					// and update oboe settings
					remainingTime := expireAt - uint32(time.Now().Unix())
					if remainingTime <= 0 {
						stdlog.Print("Updating settings from file.")
						fbw.updateSettingAndCacheFromFile()
					}
				}
			}
		}
	}()
}

func (fbw *fileBasedWatcher) Stop() {
	stdlog.Print("Stopping settings file watcher.")
	exit <- true
}