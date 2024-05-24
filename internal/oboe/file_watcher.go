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
	"math"
	"time"

	"github.com/coocood/freecache"
)

const (
	cacheSize            = 100 * 1024 // 100 kB
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
	ttlBits := math.Float64bits(float64(sl.ttl))
	ttlBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(ttlBytes, ttlBits)
	err := fbw.settingsCache.Set(keyTtl, ttlBytes, int(sl.ttl))
	if err != nil {
		stdlog.Fatalf("There was an issue with setting settingsCache: %s", err)
		// TODO: disable APM Go
	}
}

// updateSettingFromFile updates oboe settings using normalized settings from file.
func (fbw *fileBasedWatcher) updateSettingFromFile(sl *settingLambdaNormalized) {
	stdlog.Printf("TODO: Implement updateSettingFromFile. Received normalized settings %v", sl)
	// fbw.o.UpdateSetting(
	// 	sl.sType,
	// 	sl.layer,
	// 	sl.flags,
	// 	sl.value,
	// 	sl.ttl,
	// 	sl.args,
	// )
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
				// expired values are Not Found
				_, notFound := fbw.settingsCache.Get(keyTtl)
				if notFound != nil {
					stdlog.Printf("Checking settings from file.")
					fbw.updateSettingAndCacheFromFile()
				}
			}
		}
	}()
}

func (fbw *fileBasedWatcher) Stop() {
	stdlog.Print("Stopping settings file watcher.")
	exit <- true
}
