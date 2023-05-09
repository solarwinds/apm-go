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
package hdrhist

// This package doesn't actually do anything unsafe. We just
// need to import unsafe so that we can get the size of data
// types so we can return memory usage estimates to the user.
//
// Keep all uses of unsafe here so that we make sure unsafe
// is not imported in any of the other files.

import (
	"time"
	"unsafe"
)

var histSize = int(unsafe.Sizeof(Hist{}))
var timeSize = int(unsafe.Sizeof(time.Time{}) + unsafe.Sizeof(time.Location{}))
