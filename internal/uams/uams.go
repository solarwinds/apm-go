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
	"github.com/solarwinds/apm-go/internal/log"
)

var (
	uamsFilePath  = determineFileForOS()
	uamsClientURL = clientUrl
)

func readUamsClientId() (uuid.UUID, error) {
	i, err := ReadFromFile(uamsFilePath)

	if err != nil {
		log.Debugf("Could not retrieve UAMS client ID from file (reason: %s), falling back to HTTP", err)
		i, err = ReadFromHttp(uamsClientURL)
		if err != nil {
			log.Debugf("Could not retrieve UAMS client ID from HTTP (reason: %s), setting to nil default", err)
		}
	}
	return i, err
}
