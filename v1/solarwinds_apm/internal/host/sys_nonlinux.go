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
//go:build !linux

package host

// IsPhysicalInterface checks if the network interface is physical. It always
// returns true for non-Linux platforms.
func IsPhysicalInterface(ifname string) bool { return true }

// initDistro returns the ditro information of the system, it returns Unkown-not-Linux
// for non-Linux platforms.
func initDistro() string {
	return "Unknown-not-Linux"
}
