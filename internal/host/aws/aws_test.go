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

package aws

import (
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go/middleware"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwindscloud/apm-proto/go/collectorpb"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestAvailabilityZone(t *testing.T) {
	once.Do(func() {
		// no-op to avoid calling the aws sdk
	})
	memoized = nil
	require.Equal(t, "", AvailabilityZone())
	memoized = &MetadataWrapper{output: &imds.GetInstanceIdentityDocumentOutput{
		InstanceIdentityDocument: imds.InstanceIdentityDocument{
			AvailabilityZone: "my zone",
		},
	}}
	require.Equal(t, "my zone", AvailabilityZone())
}

func TestInstanceID(t *testing.T) {
	once.Do(func() {
		// no-op to avoid calling the aws sdk
	})
	memoized = nil
	require.Equal(t, "", InstanceID())
	memoized = &MetadataWrapper{output: &imds.GetInstanceIdentityDocumentOutput{
		InstanceIdentityDocument: imds.InstanceIdentityDocument{
			InstanceID: "my instance",
		},
	}}
	require.Equal(t, "my instance", InstanceID())
}

func TestMemoizeMetadata(t *testing.T) {
	// reset the metadata
	memoized = nil
	// reset the once
	once = sync.Once{}

	// we actually call the aws sdk here, and it should be nil in any tests
	require.Nil(t, MemoizeMetadata())
	// this will be wrapped by the once, so it won't hit the sdk again
	require.Nil(t, MemoizeMetadata())

	// now we manually set the metadata
	memoized = &MetadataWrapper{}
	require.NotNil(t, MemoizeMetadata())
	require.Equal(t, memoized, MemoizeMetadata())
}

func TestMetadataWrapperToPB(t *testing.T) {
	once.Do(func() {
		// no-op to avoid calling the aws sdk
	})
	memoized = &MetadataWrapper{output: &imds.GetInstanceIdentityDocumentOutput{
		InstanceIdentityDocument: imds.InstanceIdentityDocument{
			AccountID:        "my account id",
			Region:           "my region",
			AvailabilityZone: "my az",
			InstanceID:       "my host id",
			ImageID:          "my host image id",
			InstanceType:     "my host type",
		},
		ResultMetadata: middleware.Metadata{},
	}}

	expected := &collectorpb.Aws{
		CloudProvider:         "aws",
		CloudPlatform:         "aws_ec2",
		CloudAccountId:        "my account id",
		CloudRegion:           "my region",
		CloudAvailabilityZone: "my az",
		HostId:                "my host id",
		HostImageId:           "my host image id",
		HostName:              host.Hostname(),
		HostType:              "my host type",
	}
	require.Equal(t, expected, MemoizeMetadata().ToPB())
}
