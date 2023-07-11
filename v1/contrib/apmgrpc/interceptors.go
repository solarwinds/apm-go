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
	"fmt"
	"github.com/pkg/errors"
	fp "path/filepath"
	"strings"
)

func actionFromMethod(method string) string {
	mParts := strings.Split(method, "/")

	return mParts[len(mParts)-1]
}

// StackTracer is a copy of the stackTracer interface of pkg/errors.
//
// This may be fragile as stackTracer is not imported, just try our best though.
type StackTracer interface {
	StackTrace() errors.StackTrace
}

func getErrClass(err error) string {
	if st, ok := err.(StackTracer); ok {
		pkg, e := getTopFramePkg(st)
		if e == nil {
			return pkg
		}
	}
	// seems we cannot do anything else, so just return the fallback value
	return "error"
}

var (
	errNilStackTracer  = errors.New("nil stackTracer pointer")
	errEmptyStackTrace = errors.New("empty stack trace")
	errGetTopFramePkg  = errors.New("failed to get top frame package name")
)

func getTopFramePkg(st StackTracer) (string, error) {
	if st == nil {
		return "", errNilStackTracer
	}
	trace := st.StackTrace()
	if len(trace) == 0 {
		return "", errEmptyStackTrace
	}
	fs := fmt.Sprintf("%+s", trace[0])
	// it is fragile to use this hard-coded separator
	// see: https://github.com/pkg/errors/blob/30136e27e2ac8d167177e8a583aa4c3fea5be833/stack.go#L63
	frames := strings.Split(fs, "\n\t")
	if len(frames) != 2 {
		return "", errGetTopFramePkg
	}
	return fp.Base(fp.Dir(frames[1])), nil
}
