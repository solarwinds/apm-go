// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
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

package reporter

import (
	"fmt"
	"time"

	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/rand"
	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

func CreateAndSendOneTimeInitMessage(reporter Reporter, resource *resource.Resource) {
	sendInitMessage(reporter, resource)
}

func CreateInitMessage(tid trace.TraceID, r *resource.Resource) Event {
	evt := NewEventWithRandomOpID(tid, time.Now())
	evt.SetLabel(LabelUnset)
	for _, kv := range r.Attributes() {
		if kv.Key != semconv.ServiceNameKey {
			evt.AddKV(kv)
		}
	}

	evt.AddKVs([]attribute.KeyValue{
		attribute.Bool("__Init", true),
		attribute.String("APM.Version", utils.Version()),
	})
	return evt
}

func sendInitMessage(r Reporter, rsrc *resource.Resource) {
	if r.Closed() {
		log.Info(fmt.Errorf("send init message: %w", ErrReporterIsClosed))
		return
	}
	tid := trace.TraceID{0}
	rand.Random(tid[:])
	evt := CreateInitMessage(tid, rsrc)
	if err := r.ReportStatus(evt); err != nil {
		log.Error("could not send init message", err)
	}
}
