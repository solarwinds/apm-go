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
// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import errors "github.com/pkg/errors"
import mock "github.com/stretchr/testify/mock"

// StackTracer is an autogenerated mock type for the StackTracer type
type StackTracer struct {
	mock.Mock
}

// StackTrace provides a mock function with given fields:
func (_m *StackTracer) StackTrace() errors.StackTrace {
	ret := _m.Called()

	var r0 errors.StackTrace
	if rf, ok := ret.Get(0).(func() errors.StackTrace); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(errors.StackTrace)
		}
	}

	return r0
}