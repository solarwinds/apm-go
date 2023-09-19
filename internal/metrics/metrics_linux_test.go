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
	"github.com/solarwinds/apm-go/internal/bson"
	"github.com/solarwinds/apm-go/internal/utils"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendUname(t *testing.T) {
	bbuf := bson.NewBuffer()
	appendUname(bbuf)
	bbuf.Finish()
	m := bsonToMap(bbuf)

	var sysname, release string

	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err == nil {
		sysname = utils.Byte2String(uname.Sysname[:])
		release = utils.Byte2String(uname.Release[:])
		sysname = strings.TrimRight(sysname, "\x00")
		release = strings.TrimRight(release, "\x00")
	}

	assert.Equal(t, sysname, m["UnameSysName"])
	assert.Equal(t, release, m["UnameVersion"])
}
