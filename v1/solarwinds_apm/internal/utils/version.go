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

package utils

import (
	"runtime"
	"strings"
)

var (
	// The SolarWinds Observability Go agent version
	version = "1.15.0" // TODO

	// The Go version
	goVersion = strings.TrimPrefix(runtime.Version(), "go")
)

// Version returns the agent's version
func Version() string {
	return version
}

// GoVersion returns the Go version
func GoVersion() string {
	return goVersion
}
