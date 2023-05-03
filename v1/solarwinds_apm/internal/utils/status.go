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
package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	installDir        string
	installTsInSec    int64
	lastRestartInUSec int64
)

func init() {
	installDir = initInstallDir()
	installTsInSec = initInstallTsInSec()
	lastRestartInUSec = initLastRestartInUSec()
}

func initInstallDir() string {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return "unknown"
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "unknown"
	}

	prevPath := string(os.PathSeparator)
	for path != prevPath {
		base := filepath.Base(path)
		if base == "solarwinds_apm" {
			return path
		}
		prevPath = path
		path = filepath.Dir(path)
	}
	return "unknown"
}

func initInstallTsInSec() int64 {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return 0
	}
	fileStat, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fileStat.ModTime().Unix()
}

func initLastRestartInUSec() int64 {
	return time.Now().UnixNano() / 1e3
}

func InstallDir() string {
	return installDir
}

func InstallTsInSec() int64 {
	return installTsInSec
}

func LastRestartInUSec() int64 {
	return lastRestartInUSec
}
