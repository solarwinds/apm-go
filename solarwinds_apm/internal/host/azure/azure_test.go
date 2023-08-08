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
	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestQueryAzureBadStatus(t *testing.T) {
	svr := testutils.Srv(t, validJson, http.StatusInternalServerError)
	m, err := queryAzureIMDS(svr.URL)
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, "invalid status: 500; expected 200", err.Error())
}

func TestQueryAzureInvalidJson(t *testing.T) {
	svr := testutils.Srv(t, "Now is the winter of our discontent", http.StatusOK)
	m, err := queryAzureIMDS(svr.URL)
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, "failed to unmarshal json: invalid character 'N' looking for beginning of value", err.Error())
}

func TestQueryAzureNoHttpResponse(t *testing.T) {
	m, err := queryAzureIMDS("http://127.0.0.1:12345/asdf")
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, `Get "http://127.0.0.1:12345/asdf?api-version=2021-12-13&format=json": dial tcp 127.0.0.1:12345: connect: connection refused`, err.Error())
	assert.Nil(t, m.ToPB())
}

func TestQueryAzureIMDS(t *testing.T) {
	svr := testutils.Srv(t, validJson, http.StatusOK)
	m, err := queryAzureIMDS(svr.URL)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Equal(t, "westus", m.Location)
	require.Equal(t, "examplevmname", m.Name)
	require.Equal(t, "macikgo-test-may-23", m.ResourceGroupName)
	require.Equal(t, "xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx", m.SubscriptionID)
	require.Equal(t, "02aab8a4-74ef-476e-8182-f6d2ba4166a6", m.VMID)
	require.Equal(t, "crpteste9vflji9", m.VMScaleSetName)
	require.Equal(t, "Standard_A3", m.VMSize)

	// Test the conversion to protobuf we send to the collector
	pb := m.ToPB()
	require.NotNil(t, pb)
	require.Equal(t, "azure", pb.CloudProvider)
	require.Equal(t, "azure_vm", pb.CloudPlatform)
	require.Equal(t, "westus", pb.CloudRegion)
	require.Equal(t, "examplevmname", pb.HostName)
	require.Equal(t, "examplevmname", pb.AzureVmName)
	require.Equal(t, "macikgo-test-may-23", pb.AzureResourceGroupName)
	require.Equal(t, "xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx", pb.CloudAccountId)
	require.Equal(t, "02aab8a4-74ef-476e-8182-f6d2ba4166a6", pb.HostId)
	require.Equal(t, "crpteste9vflji9", pb.AzureVmScaleSetName)
	require.Equal(t, "Standard_A3", pb.AzureVmSize)
}

// Example JSON from https://learn.microsoft.com/en-us/azure/virtual-machines/instance-metadata-service
var validJson = `{
	"azEnvironment": "AZUREPUBLICCLOUD",
	"additionalCapabilities": {
		"hibernationEnabled": "true"
	},
	"hostGroup": {
	  "id": "testHostGroupId"
	}, 
	"extendedLocation": {
		"type": "edgeZone",
		"name": "microsoftlosangeles"
	},
	"evictionPolicy": "",
	"isHostCompatibilityLayerVm": "true",
	"licenseType":  "",
	"location": "westus",
	"name": "examplevmname",
	"offer": "UbuntuServer",
	"osProfile": {
		"adminUsername": "admin",
		"computerName": "examplevmname",
		"disablePasswordAuthentication": "true"
	},
	"osType": "Linux",
	"placementGroupId": "f67c14ab-e92c-408c-ae2d-da15866ec79a",
	"plan": {
		"name": "planName",
		"product": "planProduct",
		"publisher": "planPublisher"
	},
	"platformFaultDomain": "36",
	"platformSubFaultDomain": "",        
	"platformUpdateDomain": "42",
	"priority": "Regular",
	"publicKeys": [{
			"keyData": "ssh-rsa 0",
			"path": "/home/user/.ssh/authorized_keys0"
		},
		{
			"keyData": "ssh-rsa 1",
			"path": "/home/user/.ssh/authorized_keys1"
		}
	],
	"publisher": "Canonical",
	"resourceGroupName": "macikgo-test-may-23",
	"resourceId": "/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/resourceGroups/macikgo-test-may-23/providers/Microsoft.Compute/virtualMachines/examplevmname",
	"securityProfile": {
		"secureBootEnabled": "true",
		"virtualTpmEnabled": "false",
		"encryptionAtHost": "true",
		"securityType": "TrustedLaunch"
	},
	"sku": "18.04-LTS",
	"storageProfile": {
		"dataDisks": [{
			"bytesPerSecondThrottle": "979202048",
			"caching": "None",
			"createOption": "Empty",
			"diskCapacityBytes": "274877906944",
			"diskSizeGB": "1024",
			"image": {
			  "uri": ""
			},
			"isSharedDisk": "false",
			"isUltraDisk": "true",
			"lun": "0",
			"managedDisk": {
			  "id": "/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/resourceGroups/macikgo-test-may-23/providers/Microsoft.Compute/disks/exampledatadiskname",
			  "storageAccountType": "StandardSSD_LRS"
			},
			"name": "exampledatadiskname",
			"opsPerSecondThrottle": "65280",
			"vhd": {
			  "uri": ""
			},
			"writeAcceleratorEnabled": "false"
		}],
		"imageReference": {
			"id": "",
			"offer": "UbuntuServer",
			"publisher": "Canonical",
			"sku": "16.04.0-LTS",
			"version": "latest"
		},
		"osDisk": {
			"caching": "ReadWrite",
			"createOption": "FromImage",
			"diskSizeGB": "30",
			"diffDiskSettings": {
				"option": "Local"
			},
			"encryptionSettings": {
			  "enabled": "false",
			  "diskEncryptionKey": {
				"sourceVault": {
				  "id": "/subscriptions/test-source-guid/resourceGroups/testrg/providers/Microsoft.KeyVault/vaults/test-kv"
				},
				"secretUrl": "https://test-disk.vault.azure.net/secrets/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx"
			  },
			  "keyEncryptionKey": {
				"sourceVault": {
				  "id": "/subscriptions/test-key-guid/resourceGroups/testrg/providers/Microsoft.KeyVault/vaults/test-kv"
				},
				"keyUrl": "https://test-key.vault.azure.net/secrets/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx"
			  }
			},
			"image": {
				"uri": ""
			},
			"managedDisk": {
				"id": "/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/resourceGroups/macikgo-test-may-23/providers/Microsoft.Compute/disks/exampleosdiskname",
				"storageAccountType": "StandardSSD_LRS"
			},
			"name": "exampleosdiskname",
			"osType": "Linux",
			"vhd": {
				"uri": ""
			},
			"writeAcceleratorEnabled": "false"
		},
		"resourceDisk": {
			"size": "4096"
		}
	},
	"subscriptionId": "xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx",
	"tags": "baz:bash;foo:bar",
	"version": "15.05.22",
	"virtualMachineScaleSet": {
		"id": "/subscriptions/xxxxxxxx-xxxxx-xxx-xxx-xxxx/resourceGroups/resource-group-name/providers/Microsoft.Compute/virtualMachineScaleSets/virtual-machine-scale-set-name"
	},
	"vmId": "02aab8a4-74ef-476e-8182-f6d2ba4166a6",
	"vmScaleSetName": "crpteste9vflji9",
	"vmSize": "Standard_A3",
	"zone": ""
}`
