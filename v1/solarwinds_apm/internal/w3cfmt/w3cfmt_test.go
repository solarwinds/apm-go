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
package w3cfmt

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

const spanIdHex = "0123456789abcdef"

var spanId, err = trace.SpanIDFromHex(spanIdHex)

func init() {
	if err != nil {
		log.Fatal("Fatal error: ", err)
	}
}

func TestSwFromCtx(t *testing.T) {
	sc := trace.SpanContext{}.WithSpanID(spanId).WithTraceFlags(trace.TraceFlags(0x00))

	assert.Equal(t, fmt.Sprintf("%s-00", spanIdHex), SwFromCtx(sc))

	sc = sc.WithTraceFlags(trace.TraceFlags(0x01))
	assert.Equal(t, fmt.Sprintf("%s-01", spanIdHex), SwFromCtx(sc))
}
