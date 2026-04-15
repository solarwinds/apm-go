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

package swo

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/host/k8s"
	"github.com/solarwinds/apm-go/internal/instance"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/uams"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func createResource(resourceAttrs ...attribute.KeyValue) (*resource.Resource, error) {
	if serviceKey, ok := config.ParsedServiceKey(); ok && os.Getenv(config.EnvOtelServiceNameKey) == "" {
		if err := os.Setenv(config.EnvOtelServiceNameKey, serviceKey.ServiceName); err != nil {
			log.Warningf("could not override unset environment variable %s based on service key, err: %s", config.EnvOtelServiceNameKey, err)
		}
	}

	if os.Getenv(config.EnvEnableExperimentalDetector) == "" {
		if err := os.Setenv(config.EnvEnableExperimentalDetector, "false"); err != nil {
			log.Warningf("could not override unset environment variable %s, err: %s", config.EnvEnableExperimentalDetector, err)
		}
	}

	customResource, customResourceErrors := resource.New(context.Background(),
		resource.WithContainer(),
		resource.WithOS(),
		resource.WithProcess(),
		// Process runtime description is not recommended[1] for Go and thus is not added by `WithProcess` above.
		// Example value: go version go1.20.4 linux/arm64
		// [1]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/process.md#go-runtimes
		resource.WithProcessRuntimeDescription(),
		resource.WithDetectors(getOptionalDetectors()...),
		instance.WithInstanceDetector(),
		resource.WithAttributes(resourceAttrs...),
		resource.WithAttributes(attribute.String("sw.data.module", "apm")),
		resource.WithAttributes(attribute.String("sw.apm.version", utils.Version())),
	)
	r, mergedError := resource.Merge(resource.Default(), customResource)
	combined := errors.Join(customResourceErrors, mergedError)
	if errors.Is(combined, resource.ErrSchemaURLConflict) {
		// ErrSchemaURLConflict is non-fatal: it signals a semconv version mismatch
		// between detector libraries. The resource attributes are still detected
		// correctly. Log it as a warning and strip it from the returned error.
		// The OTel SDK's own ExampleNew() treats this error as non-fatal by only
		// logging it:
		// https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/resource/example_test.go
		log.Warningf("resource schema URL conflict (possible detector library version mismatch): %v", combined)
		combined = filterSchemaURLConflict(combined)
	}
	return r, combined
}

// filterSchemaURLConflict removes ErrSchemaURLConflict from a joined error,
// returning nil when the conflict was the only error present.
func filterSchemaURLConflict(combined error) error {
	type multiErr interface{ Unwrap() []error }
	if u, ok := combined.(multiErr); ok {
		var remaining []error
		for _, e := range u.Unwrap() {
			if !errors.Is(e, resource.ErrSchemaURLConflict) {
				remaining = append(remaining, e)
			}
		}
		return errors.Join(remaining...)
	}
	return nil
}

func getOptionalDetectors() []resource.Detector {
	disabledResouceDetectors := os.Getenv(config.EnvSolarwindsDisabledResourceDetectors)

	optionalDetectors := []resource.Detector{}
	if !strings.Contains(disabledResouceDetectors, "uams") {
		optionalDetectors = append(optionalDetectors, uams.New())
	}
	if !strings.Contains(disabledResouceDetectors, "ec2") {
		optionalDetectors = append(optionalDetectors, ec2.NewResourceDetector())
	}
	if !strings.Contains(disabledResouceDetectors, "azurevm") {
		optionalDetectors = append(optionalDetectors, azurevm.New())
	}
	if !strings.Contains(disabledResouceDetectors, "k8s") {
		optionalDetectors = append(optionalDetectors, k8s.NewResourceDetector())
	}

	return optionalDetectors
}
