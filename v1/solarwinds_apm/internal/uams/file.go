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
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"os"
)

const linuxFilePath = "/opt/solarwinds/uamsclient/var/uamsclientid"
const windowsFilePath = `C:\ProgramData\SolarWinds\UAMSClient\uamsclientid`

func ReadFromFile(f string) (uuid.UUID, error) {
	if st, err := os.Stat(f); err != nil {
		return uuid.Nil, errors.Wrap(err, "could not stat uams client file")
	} else if st.IsDir() {
		return uuid.Nil, fmt.Errorf("could not open path (%s); Expected a file, got a directory instead", f)
	}

	if data, err := os.ReadFile(f); err != nil {
		return uuid.Nil, errors.Wrap(err, fmt.Sprintf("could not read uams client file (%s)", f))
	} else {
		if uid, err := uuid.ParseBytes(data); err != nil {
			return uuid.Nil, errors.Wrap(err, fmt.Sprintf("uams client file (%s) did not contain parseable UUID", f))
		} else {
			return uid, nil
		}
	}
}
