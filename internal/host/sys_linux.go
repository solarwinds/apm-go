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
package host

import (
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

// IsPhysicalInterface returns true if the specified interface name is physical
func IsPhysicalInterface(ifname string) bool {
	fn := filepath.Join("/sys/class/net/", ifname)
	link, err := os.Readlink(fn)
	if err != nil {
		log.Infof("cannot read link %s", fn)
		return true
	}
	if strings.Contains(link, "/virtual/") {
		return false
	}
	return true
}
