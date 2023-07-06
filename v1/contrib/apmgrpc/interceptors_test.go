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
package solarwinds_apmgrpc

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/contrib/apmgrpc/mocks"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm"
	"github.com/stretchr/testify/assert"
)

func TestGetTopFramePkg(t *testing.T) {
	// nil pointer
	pkg, err := getTopFramePkg(nil)
	assert.Equal(t, "", pkg)
	assert.NotNil(t, err)

	// returns nil
	m := mocks.StackTracer{}
	m.On("StackTrace").Return(nil)
	pkg, err = getTopFramePkg(&m)
	assert.Equal(t, "", pkg)
	assert.Equal(t, errEmptyStackTrace.Error(), err.Error())

	// returns empty frame stack
	m = mocks.StackTracer{}
	m.On("StackTrace").Return(errors.StackTrace{})
	pkg, err = getTopFramePkg(&m)
	assert.Equal(t, "", pkg)
	assert.Equal(t, errEmptyStackTrace.Error(), err.Error())

	// error from this package
	e := errors.Wrap(errors.New("inner error"), "wrapper")
	if ste, ok := e.(StackTracer); ok {
		pkg, err = getTopFramePkg(ste)
		assert.Equal(t, "apmgrpc", pkg)
		assert.Nil(t, err)

		assert.Equal(t, "apmgrpc", getErrClass(e))
	} else {
		assert.Equal(t, "error", getErrClass(e))
	}

	// error from another package
	e = solarwinds_apm.SetLogLevel("invalid_level")
	if ste, ok := e.(StackTracer); ok {
		pkg, err = getTopFramePkg(ste)
		assert.Equal(t, "solarwinds_apm", pkg)
		assert.Nil(t, err)

		assert.Equal(t, "solarwinds_apm", getErrClass(e))
	} else {
		assert.Equal(t, "error", getErrClass(e))
	}
}

func TestActionFromMethod(t *testing.T) {
	assert.EqualValues(t, "b", actionFromMethod("a/b"))
	assert.EqualValues(t, "c", actionFromMethod("a/b/c"))
	assert.EqualValues(t, "abc", actionFromMethod("abc"))
	assert.EqualValues(t, "", actionFromMethod(""))
	assert.EqualValues(t, "abc", actionFromMethod("/abc"))
	assert.EqualValues(t, "", actionFromMethod("abc/"))
	assert.EqualValues(t, "", actionFromMethod("/abc/"))
}
