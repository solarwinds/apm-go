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

package k8s

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"testing"
)

// Test requestMetadata

func TestRequestMetadataFromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("SW_K8S_POD_NAMESPACE", ""))
	md, err := requestMetadata()
	require.Error(t, err)
	require.Nil(t, md)
	require.Equal(t, "k8s namespace was empty", err.Error())

	require.NoError(t, os.Setenv("SW_K8S_POD_NAMESPACE", "my env namespace"))
	defer func() {
		require.NoError(t, os.Unsetenv("SW_K8S_POD_NAMESPACE"))
	}()
	md, err = requestMetadata()
	require.NoError(t, err)
	var hostname string
	hostname, err = os.Hostname()
	require.NoError(t, err)
	require.NotNil(t, md)
	require.Equal(t, "my env namespace", md.Namespace)
	require.Equal(t, hostname, md.PodName)
	require.Equal(t, "", md.PodUid)

	require.NoError(t, os.Setenv("SW_K8S_POD_NAME", "my env pod name"))
	defer func() {
		require.NoError(t, os.Unsetenv("SW_K8S_POD_NAME"))
	}()

	require.NoError(t, os.Setenv("SW_K8S_POD_UID", "my env uid"))
	defer func() {
		require.NoError(t, os.Unsetenv("SW_K8S_POD_UID"))
	}()

	md, err = requestMetadata()
	require.NoError(t, err)
	require.NotNil(t, md)
	require.Equal(t, "my env namespace", md.Namespace)
	require.Equal(t, "my env pod name", md.PodName)
	require.Equal(t, "my env uid", md.PodUid)
}

func TestRequestMetadataNoNamespace(t *testing.T) {
	md, err := requestMetadata()
	require.Error(t, err)
	require.Nil(t, md)
	require.Equal(t, fmt.Sprintf("open %s: no such file or directory", determineNamspaceFileForOS()), err.Error())
}

func TestMetadata_ToPB(t *testing.T) {
	md := &Metadata{
		Namespace: "foo",
		PodName:   "bar",
		PodUid:    "baz",
	}
	pb := md.ToPB()
	require.Equal(t, md.Namespace, pb.Namespace)
	require.Equal(t, md.PodName, pb.PodName)
	require.Equal(t, md.PodUid, pb.PodUid)
}

// Test getNamespace

func TestGetNamespaceFromFallbackFile(t *testing.T) {
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(f.Name()))
	}()
	_, err = f.WriteString("my file namespace")
	require.NoError(t, err)
	ns, err := getNamespace(f.Name())
	require.NoError(t, err)
	require.Equal(t, "my file namespace", ns)
}

func TestGetNamespaceFromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("SW_K8S_POD_NAMESPACE", "my env namespace"))
	defer func() {
		require.NoError(t, os.Unsetenv("SW_K8S_POD_NAMESPACE"))
	}()
	ns, err := getNamespace("this file does not exist and should not be opened")
	require.NoError(t, err)
	require.Equal(t, "my env namespace", ns)
}

func TestGetNamespaceNoneFound(t *testing.T) {
	require.NoError(t, os.Unsetenv("SW_K8S_POD_NAMESPACE"))
	ns, err := getNamespace("this file does not exist and should not be opened")
	require.Error(t, err)
	require.Equal(t, "open this file does not exist and should not be opened: no such file or directory", err.Error())
	require.Equal(t, "", ns)

}

// Test getPodName

func TestGetPodNameHostname(t *testing.T) {
	pn, err := getPodname()
	require.NoError(t, err)
	hostname, err := os.Hostname()
	require.NoError(t, err)
	require.Equal(t, hostname, pn)
}

func TestGetPodNameFromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("SW_K8S_POD_NAME", "my env pod name"))
	defer func() {
		require.NoError(t, os.Unsetenv("SW_K8S_POD_NAME"))
	}()
	pn, err := getPodname()
	require.NoError(t, err)
	require.Equal(t, "my env pod name", pn)
}

// Test getPodUid

func TestGetPodUidFromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("SW_K8S_POD_UID", "0c04997a-a33e-44d6-8185-32fb1cb4357f"))
	defer func() {
		require.NoError(t, os.Unsetenv("SW_K8S_POD_UID"))
	}()
	uid, err := getPodUid("fallback file does not exist")
	require.NoError(t, err)
	require.Equal(t, "0c04997a-a33e-44d6-8185-32fb1cb4357f", uid)
}

func TestGetPodUidFromFileFails(t *testing.T) {
	require.NoError(t, os.Unsetenv("SW_K8S_POD_UID"))
	uid, err := getPodUid(linuxProcMountInfo)
	require.Error(t, err)
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "linux" {
		require.Equal(t, "no match found in file /proc/self/mountinfo", err.Error())
	} else {
		require.Equal(t, "cannot determine k8s pod uid on host OS", err.Error())
	}
	require.Equal(t, "", uid)
}

func TestGetPodUidFromProc(t *testing.T) {
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(f.Name()))
	}()
	_, err = f.WriteString(sampleProc)
	require.NoError(t, err)
	uid, err := getPodUidFromProc(f.Name())
	require.NoError(t, err)
	require.Equal(t, "9dcdb600-4156-4b7b-afcc-f8c06cb0e474", uid)
}

func TestGetPodUidFromProcNoMatch(t *testing.T) {
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(f.Name()))
	}()
	_, err = f.WriteString(`
just a file that should not match the regex
`)
	require.NoError(t, err)
	uid, err := getPodUidFromProc(f.Name())
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("no match found in file %s", f.Name()), err.Error())
	require.Equal(t, "", uid)
}

var sampleProc = `5095 3607 0:432 / / ro,relatime master:975 - overlay overlay rw,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2416/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2415/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2414/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2413/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2412/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2411/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2410/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2409/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2408/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2417/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/2417/work,xino=off
5096 5095 0:433 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
5097 5095 0:434 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
5098 5097 0:435 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
5099 5097 0:402 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
5100 5095 0:407 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
5101 5100 0:436 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
5102 5101 0:30 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime master:11 - cgroup cgroup rw,xattr,name=systemd
5103 5101 0:33 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime master:15 - cgroup cgroup rw,blkio
5130 5101 0:34 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/cpu,cpuacct ro,nosuid,nodev,noexec,relatime master:16 - cgroup cgroup rw,cpu,cpuacct
5131 5101 0:35 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime master:17 - cgroup cgroup rw,devices
5132 5101 0:36 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime master:18 - cgroup cgroup rw,freezer
5133 5101 0:37 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime master:19 - cgroup cgroup rw,memory
5134 5101 0:38 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime master:20 - cgroup cgroup rw,cpuset
5162 5101 0:39 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime master:21 - cgroup cgroup rw,hugetlb
5163 5101 0:40 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/net_cls,net_prio ro,nosuid,nodev,noexec,relatime master:22 - cgroup cgroup rw,net_cls,net_prio
5164 5101 0:41 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime master:23 - cgroup cgroup rw,pids
5165 5101 0:42 / /sys/fs/cgroup/rdma ro,nosuid,nodev,noexec,relatime master:24 - cgroup cgroup rw,rdma
5166 5101 0:43 /kubepods/burstable/pod9dcdb600-4156-4b7b-afcc-f8c06cb0e474/b92a703b826d4494978d810adf100d06c5ade2539b714d984ae3c8fef3449964 /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime master:25 - cgroup cgroup rw,perf_event
5167 5095 8:1 /var/lib/kubelet/pods/9dcdb600-4156-4b7b-afcc-f8c06cb0e474/volumes/kubernetes.io~empty-dir/temp-dir /tmp rw,relatime - ext4 /dev/sda1 rw,discard
5175 5095 8:1 /var/lib/kubelet/pods/9dcdb600-4156-4b7b-afcc-f8c06cb0e474/etc-hosts /etc/hosts rw,relatime - ext4 /dev/sda1 rw,discard
5176 5097 8:1 /var/lib/kubelet/pods/9dcdb600-4156-4b7b-afcc-f8c06cb0e474/containers/trace-hopper/12b93bce /dev/termination-log rw,relatime - ext4 /dev/sda1 rw,discard
5177 5095 8:1 /var/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/a079f12d3215697a69dbb86b58dd6d6957c54c8588b9dfa6b210e22ec70f279e/hostname /etc/hostname ro,relatime - ext4 /dev/sda1 rw,discard
5181 5095 8:1 /var/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/a079f12d3215697a69dbb86b58dd6d6957c54c8588b9dfa6b210e22ec70f279e/resolv.conf /etc/resolv.conf ro,relatime - ext4 /dev/sda1 rw,discard
5189 5097 0:400 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
5203 5095 8:1 /var/lib/kubelet/pods/9dcdb600-4156-4b7b-afcc-f8c06cb0e474/volumes/kubernetes.io~configmap/trace-hopper-vol/..2023_05_11_12_36_52.1754749256/trace-hopper.yaml /app/resources/trace-hopper.yaml ro,relatime - ext4 /dev/sda1 rw,discard
5204 5095 8:1 /var/lib/kubelet/pods/9dcdb600-4156-4b7b-afcc-f8c06cb0e474/volumes/kubernetes.io~configmap/trace-hopper-vol/..2023_05_11_12_36_52.1754749256/javaagent.json /app/resources/javaagent.json ro,relatime - ext4 /dev/sda1 rw,discard
5205 5095 0:399 / /run/secrets/kubernetes.io/serviceaccount ro,relatime - tmpfs tmpfs rw,size=4954828k
3608 5096 0:433 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
3609 5096 0:433 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
3610 5096 0:433 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
3611 5096 0:433 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
3676 5096 0:433 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
3677 5096 0:437 / /proc/acpi ro,relatime - tmpfs tmpfs ro
3679 5096 0:434 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
3680 5096 0:434 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
3681 5096 0:434 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
3682 5096 0:434 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
3683 5096 0:438 / /proc/scsi ro,relatime - tmpfs tmpfs ro
3684 5100 0:452 / /sys/firmware ro,relatime - tmpfs tmpfs ro`
