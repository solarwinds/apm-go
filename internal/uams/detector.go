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

package uams

import (
	"context"

	"github.com/solarwinds/apm-go/internal/constants"
	"github.com/solarwinds/apm-go/internal/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type resourceDetector struct {
}

func New() *resourceDetector {
	return &resourceDetector{}
}

func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	id, err := readUamsClientId()
	if err != nil {
		log.Info("Uams Client Id not detected, ", err)
		return nil, nil
	}

	attributes := []attribute.KeyValue{
		attribute.String(constants.UamsClientIdAttribute, id.String()),
		semconv.HostID(id.String()),
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}
