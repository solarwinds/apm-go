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
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/uams"
	"github.com/solarwinds/apm-go/internal/utils"
	"go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
	"go.opentelemetry.io/contrib/detectors/azure/azureappservice"
	"go.opentelemetry.io/contrib/detectors/azure/azurefunctions"
	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

// createResource builds the OTel Resource, resolving service.name with the following
// precedence from low to high (later options win ties, see resource.New):
//  1. SDK default fallback `unknown_service:<executable_name>` (resource.WithService)
//  2. Other resource detectors: container, host, process, cloud (ec2, azurevm, azurefunctions), k8s, uams
//  3. SW APM service key, service name portion
//  4. Resource detector for Azure App Service
//  5. OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES env vars (resource.WithFromEnv)
//  6. Programmatic resourceAttrs passed to swo.Start
//
// Declarative configuration is intentionally not handled here.
func createResource(resourceAttrs ...attribute.KeyValue) (*resource.Resource, error) {
	if os.Getenv(config.EnvEnableExperimentalDetector) == "" {
		if err := os.Setenv(config.EnvEnableExperimentalDetector, "false"); err != nil {
			log.Warningf("could not override unset environment variable %s, err: %s", config.EnvEnableExperimentalDetector, err)
		}
	}

	customResource, customResourceErrors := resource.New(context.Background(),
		// WithService sets service.instance.id (random UUID) and a service.name fallback.
		resource.WithService(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithOS(),
		resource.WithProcess(),
		// Process runtime description is not recommended[1] for Go and thus is not added by `WithProcess` above.
		// Example value: go version go1.20.4 linux/arm64
		// [1]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/process.md#go-runtimes
		resource.WithProcessRuntimeDescription(),
		// Get service name from resource detectors (container, host, process, cloud detectors, etc.)
		resource.WithDetectors(getOtherDetectors()...),
		// The SW APM service key's service name is always present (its token is required for auth), so it
		// must yield to more specific automatic sources like the Azure App Service detector below.
		resource.WithAttributes(serviceKeyServiceNameAttrs()...),
		resource.WithDetectors(getAzureAppServiceDetector()...),
		// OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES take precedence over all automatic sources above.
		resource.WithFromEnv(),
		// Programmatic resourceAttrs passed to swo.Start take precedence over everything else.
		resource.WithAttributes(resourceAttrs...),
		resource.WithAttributes(attribute.String("sw.data.module", "apm")),
		resource.WithAttributes(attribute.String("sw.apm.version", utils.Version())),
	)
	r, mergedError := resource.Merge(resource.Default(), customResource)
	combinedError := errors.Join(customResourceErrors, mergedError)
	if errors.Is(combinedError, resource.ErrSchemaURLConflict) {
		// ErrSchemaURLConflict is non-fatal: it signals a semconv version mismatch
		// between detector libraries. The resource attributes are still detected
		// correctly. Log it as a warning and strip it from the returned error.
		// The OTel SDK's own ExampleNew() treats this error as non-fatal by only
		// logging it:
		// https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/resource/example_test.go
		log.Warningf("resource schema URL conflict (possible detector library version mismatch): %v", combinedError)
		combinedError = filterSchemaURLConflict(combinedError)
	}
	return r, combinedError
}

// filterSchemaURLConflict removes ErrSchemaURLConflict from a joined error,
// returning nil when the conflict was the only error present.
func filterSchemaURLConflict(combinedError error) error {
	type multiErr interface{ Unwrap() []error }
	if u, ok := combinedError.(multiErr); ok {
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

// serviceKeyServiceNameAttrs returns the service.name attribute derived from the SW APM
// service key, when a valid one is configured.
func serviceKeyServiceNameAttrs() []attribute.KeyValue {
	if serviceKey, ok := config.ParsedServiceKey(); ok {
		return []attribute.KeyValue{attribute.String("service.name", serviceKey.ServiceName)}
	}
	return nil
}

// getAzureAppServiceDetector returns the Azure App Service resource detector on its own so it
// can be applied with higher precedence than the SW APM service key.
// It use WEBSITE_SITE_NAME as the service name.
func getAzureAppServiceDetector() []resource.Detector {
	disabledResourceDetectors := os.Getenv(config.EnvSolarwindsDisabledResourceDetectors)
	if strings.Contains(disabledResourceDetectors, "azureappservice") {
		return nil
	}
	return []resource.Detector{azureappservice.NewResourceDetector()}
}

// getOtherDetectors returns the remaining optional resource detectors (container/host/process
// info aside), all of which rank below the SW APM service key.
func getOtherDetectors() []resource.Detector {
	disabledResourceDetectors := os.Getenv(config.EnvSolarwindsDisabledResourceDetectors)

	optionalDetectors := []resource.Detector{}
	if !strings.Contains(disabledResourceDetectors, "uams") {
		optionalDetectors = append(optionalDetectors, uams.New())
	}
	if !strings.Contains(disabledResourceDetectors, "ec2") {
		optionalDetectors = append(optionalDetectors, ec2.NewResourceDetector())
	}
	if !strings.Contains(disabledResourceDetectors, "azurevm") {
		optionalDetectors = append(optionalDetectors, azurevm.New())
	}
	if !strings.Contains(disabledResourceDetectors, "azurefunctions") {
		optionalDetectors = append(optionalDetectors, azurefunctions.NewResourceDetector())
	}
	if !strings.Contains(disabledResourceDetectors, "k8s") {
		optionalDetectors = append(optionalDetectors, k8s.NewResourceDetector())
	}

	return optionalDetectors
}
