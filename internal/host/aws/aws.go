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
	"context"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwindscloud/apm-proto/go/collectorpb"
	"sync"
)

type MetadataWrapper struct {
	output *imds.GetInstanceIdentityDocumentOutput
}

var (
	memoized *MetadataWrapper
	once     sync.Once
)

func MemoizeMetadata() *MetadataWrapper {
	once.Do(func() {
		client := imds.New(imds.Options{
			Retryer: retry.NewStandard(func(options *retry.StandardOptions) {
				// Even if we set `Backoff` here, the `imds.New` call overrides
				// it, so best we can do is modify the `MaxAttempts`. If a user
				// wishes to disable this client, the AWS SDK checks the
				// `AWS_EC2_METADATA_DISABLED` environment variable for the
				// value "true".
				options.MaxAttempts = 2
			}),
		})
		if output, err := client.GetInstanceIdentityDocument(context.Background(), nil); err != nil {
			log.Debugf("Could not retrieve aws metadata %s", err)
		} else {
			memoized = &MetadataWrapper{output}
		}
	})
	return memoized
}

func InstanceID() string {
	if md := MemoizeMetadata(); md != nil {
		return md.output.InstanceID
	}
	return ""
}

func AvailabilityZone() string {
	if md := MemoizeMetadata(); md != nil {
		return md.output.AvailabilityZone
	}
	return ""
}

func (md *MetadataWrapper) ToPB() *collectorpb.Aws {
	return &collectorpb.Aws{
		CloudProvider:         "aws",
		CloudPlatform:         "aws_ec2",
		CloudAccountId:        md.output.AccountID,
		CloudRegion:           md.output.Region,
		CloudAvailabilityZone: md.output.AvailabilityZone,
		HostId:                md.output.InstanceID,
		HostImageId:           md.output.ImageID,
		HostName:              host.Hostname(),
		HostType:              md.output.InstanceType,
	}
}
