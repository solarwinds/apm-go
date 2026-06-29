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

package swo

import (
	"sync"

	"github.com/solarwinds/apm-go/internal/oboe"
)

// globalOboe holds the oboe instance used for WaitForReady after Start() is called.
// Access is guarded by globalOboeMu to allow safe concurrent reads/writes.
var (
	globalOboeMu sync.RWMutex
	globalOboe   oboe.Oboe
)

// setGlobalOboe stores the oboe instance used by WaitForReady. Pass nil to clear it on shutdown.
func setGlobalOboe(o oboe.Oboe) {
	globalOboeMu.Lock()
	defer globalOboeMu.Unlock()
	globalOboe = o
}

// getGlobalOboe returns the oboe instance set by the most recent Start(), or nil if unset/shutdown.
func getGlobalOboe() oboe.Oboe {
	globalOboeMu.RLock()
	defer globalOboeMu.RUnlock()
	return globalOboe
}
