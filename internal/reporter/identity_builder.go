package reporter

import (
	"github.com/google/uuid"
	"github.com/solarwinds/apm-go/internal/host"
	"github.com/solarwinds/apm-go/internal/host/aws"
	"github.com/solarwinds/apm-go/internal/host/azure"
	"github.com/solarwinds/apm-go/internal/host/k8s"
	"github.com/solarwinds/apm-go/internal/log"
	"github.com/solarwinds/apm-go/internal/uams"

	collector "github.com/solarwinds/apm-proto/go/collectorpb"
)

// buildIdentity builds the HostID struct from current host metadata
func buildIdentity() *collector.HostID {
	return newHostID(host.CurrentID())
}

// buildBestEffortIdentity builds the HostID with the best effort
func buildBestEffortIdentity() *collector.HostID {
	hid := newHostID(host.BestEffortCurrentID())
	hid.Hostname = host.Hostname()
	return hid
}

func newHostID(id *host.ID) *collector.HostID {
	gid := &collector.HostID{}

	gid.Hostname = id.Hostname()

	gid.Pid = int32(id.Pid())
	gid.Ec2InstanceID = aws.InstanceID()
	gid.Ec2AvailabilityZone = aws.AvailabilityZone()
	gid.DockerContainerID = id.ContainerId()
	gid.MacAddresses = id.MAC()
	gid.HerokuDynoID = id.HerokuId()
	gid.AzAppServiceInstanceID = id.AzureAppInstId()
	gid.Uuid = id.InstanceID()
	gid.HostType = collector.HostType_PERSISTENT
	if uid := uams.GetCurrentClientId(); uid != uuid.Nil {
		gid.UamsClientID = uid.String()
	}
	if md := azure.MemoizeMetadata(); md != nil {
		gid.AzureMetadata = md.ToPB()
		log.Debugf("sending azure metadata %+v", gid.AzureMetadata)
	}
	if md := k8s.MemoizeMetadata(); md != nil {
		gid.K8SMetadata = md.ToPB()
		log.Debugf("sending k8s metadata %+v", gid.K8SMetadata)
	}
	if md := aws.MemoizeMetadata(); md != nil {
		gid.AwsMetadata = md.ToPB()
		log.Debugf("sending aws metadata %+v", gid.AwsMetadata)
	}

	return gid
}
