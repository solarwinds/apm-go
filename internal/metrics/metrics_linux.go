// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//go:build linux

package metrics

import (
	"strconv"
	"strings"

	"github.com/solarwinds/apm-go/internal/utils"
)

type linuxHostMetrics struct{}

func getHostMetrics() HostMetrics {
	return &linuxHostMetrics{}
}

func (hm *linuxHostMetrics) getTotalRAM() (uint64, bool) {
	if s := utils.GetStrByKeyword("/proc/meminfo", "MemTotal"); s != "" {
		memTotal := strings.Fields(s) // MemTotal: 7657668 kB
		if len(memTotal) == 3 {
			if total, err := strconv.Atoi(memTotal[1]); err == nil {
				return uint64(total * 1024), true
			}
		}
	}
	return 0, false
}

func (hm *linuxHostMetrics) getFreeRAM() (uint64, bool) {
	if s := utils.GetStrByKeyword("/proc/meminfo", "MemFree"); s != "" {
		memFree := strings.Fields(s) // MemFree: 161396 kB
		if len(memFree) == 3 {
			if free, err := strconv.Atoi(memFree[1]); err == nil {
				return uint64(free * 1024), true
			}
		}
	}
	return 0, false
}

func (hm *linuxHostMetrics) getSystemLoad1() (float64, bool) {
	if s := utils.GetStrByKeyword("/proc/loadavg", ""); s != "" {
		load, err := strconv.ParseFloat(strings.Fields(s)[0], 64)
		if err == nil {
			return load, true
		}
	}
	return 0, false
}
