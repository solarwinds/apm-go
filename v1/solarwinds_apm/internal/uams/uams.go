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
	"sync"
	"time"
)

var exit = make(chan bool, 1)

var currState = &state{}

type state struct {
	m sync.Mutex

	// Only valid when an update occurred
	clientId uuid.UUID
	// Time of last update
	updated time.Time
	// Updated via "file", "http" or ""
	via string
}

func Start() {
	go clientIdCheck()
	// Poll for UAMS client update once per hour
	ticker := time.NewTicker(time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-exit:
				return
			case <-ticker.C:
				log.Debug("Checking for UAMS client update")
				clientIdCheck()
			}
		}
	}()
}

func clientIdCheck() {
	f := determineFileForOS()
	i, err := ReadFromFile(f)
	if err != nil {
		log.Debugf("Could not retrieve UAMS client ID from file (reason: %s), falling back to HTTP", err)
		i, err = ReadFromHttp(clientUrl)
		if err != nil {
			log.Debugf("Could not retrieve UAMS client ID from HTTP (reason: %s), setting to nil default", err)
		} else {
			updateClientId(i, "http")
		}
	} else {
		updateClientId(i, "file")
	}
}

func updateClientId(uid uuid.UUID, via string) {
	currState.m.Lock()
	defer currState.m.Unlock()

	if uid == currState.clientId {
		log.Debug("Found the same UAMS client ID that we had before, skipping")
		return
	}

	log.Debugf("UAMS client ID (%s) successfully loaded via %s", uid, via)
	currState.clientId = uid
	currState.via = via
	currState.updated = time.Now()
}

func GetCurrentClientId() uuid.UUID {
	currState.m.Lock()
	defer currState.m.Unlock()
	return currState.clientId
}

func Stop() {
	log.Debug("Stopping UAMS client ID poller")
	exit <- true
}
