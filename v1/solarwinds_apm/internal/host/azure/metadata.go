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

package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	collector "github.com/solarwindscloud/apm-proto/go/collectorpb"
	"io"
	"net/http"
	"time"
)

type Compute struct {
	Location          string `json:"location"`
	Name              string `json:"name"`
	ResourceGroupName string `json:"resourceGroupName"`
	SubscriptionID    string `json:"subscriptionId"`
	VMID              string `json:"vmId"`
	VMScaleSetName    string `json:"vmScaleSetName"`
	VMSize            string `json:"vmSize"`
}

type Metadata struct {
	Compute Compute `json:"compute"`
}

func (m *Metadata) ToPB() *collector.Azure {
	if m == nil {
		return nil
	}
	// Mimicking the specified behavior here:
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor#azure
	return &collector.Azure{
		CloudProvider:          "azure",
		CloudPlatform:          "azure_vm",
		CloudRegion:            m.Compute.Location,
		CloudAccountId:         m.Compute.SubscriptionID,
		HostId:                 m.Compute.VMID,
		HostName:               m.Compute.Name,
		AzureVmName:            m.Compute.Name,
		AzureVmSize:            m.Compute.VMSize,
		AzureVmScaleSetName:    m.Compute.VMScaleSetName,
		AzureResourceGroupName: m.Compute.ResourceGroupName,
	}
}

const metadataUrl = "http://169.254.169.254/metadata/instance"

func RequestMetadata() (*Metadata, error) {
	return queryAzureIMDS(metadataUrl)
}

func queryAzureIMDS(url_ string) (*Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url_, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", "2021-12-13")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Metadata", "True")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if res.Body != nil {
			_ = res.Body.Close()
		}
	}()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status: %d; expected %d", res.StatusCode, http.StatusOK)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	m := &Metadata{}
	if err = json.Unmarshal(b, m); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal json")
	}
	return m, err
}
