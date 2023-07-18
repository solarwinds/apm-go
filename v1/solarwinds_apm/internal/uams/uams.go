// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
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

package uams

import (
	"github.com/google/uuid"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"runtime"
	"sync"
	"time"
)

var uamsState = &state{}

type state struct {
	m sync.Mutex

	// Only valid when an update occurred
	clientId uuid.UUID
	// Time of last update
	updated time.Time
	// Updated via "file", "http" or ""
	via string
}

func init() {
	f := linuxFilePath
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		f = winFilePath
	}
	i, err := ReadFromFile(f)
	if err != nil {
		log.Infof("Could not retrieve UAMS client ID from file (reason: %s), falling back to HTTP", err)
		i, err = ReadFromHttp(uamsClientUrl)
		if err != nil {
			log.Infof("Could not retrieve UAMS client ID from HTTP (reason: %s), setting to nil default")
		} else {
			updateClientId(i, "http")
		}
	} else {
		updateClientId(i, "file")
	}
}

func updateClientId(uid uuid.UUID, via string) {
	log.Infof("UAMS client ID (%s) successfully loaded via %s", uid, via)
	uamsState.m.Lock()
	defer uamsState.m.Unlock()

	uamsState.clientId = uid
	uamsState.via = via
	uamsState.updated = time.Now()
}

func GetCurrentClientId() uuid.UUID {
	uamsState.m.Lock()
	defer uamsState.m.Unlock()
	return uamsState.clientId
}
