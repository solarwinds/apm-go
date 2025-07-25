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

package instance

import (
	"context"

	"github.com/solarwinds/apm-go/internal/swotel/semconv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

type instanceDetector struct {
}

func WithInstanceDetector() resource.Option {
	return resource.WithDetectors(instanceDetector{})
}

func (detector instanceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		{Key: semconv.ServiceInstanceIDKey, Value: attribute.StringValue(InstanceID())},
	}
	return resource.NewSchemaless(attributes...), nil
}
