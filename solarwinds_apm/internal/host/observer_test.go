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
package host

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/solarwindscloud/solarwinds-apm-go/solarwinds_apm/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertContainerId(t *testing.T, filetext string, expectedContainerId string) {
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(f.Name()))
	}()
	_, err = f.WriteString(filetext)
	require.NoError(t, err)

	cid := getContainerIdFromFile(f.Name())
	require.Equal(t, expectedContainerId, cid)
	if expectedContainerId != "" {
		require.Len(t, cid, 64)
		_, err = hex.DecodeString(cid)
		require.NoError(t, err)
	}
}

func TestGetContainerId(t *testing.T) {
	assertContainerId(
		t,
		"12:hugetlb:/docker/c90cc641f43e3276534283717b4d08385ea0cfbcf5c7b308701d810a1bd19036",
		"c90cc641f43e3276534283717b4d08385ea0cfbcf5c7b308701d810a1bd19036",
	)

	assertContainerId(
		t,
		"12:hugetlb:/something/other:c90cc641f43e3276534283717b4d08385ea0cfbcf5c7b308701d810a1bd19036",
		"c90cc641f43e3276534283717b4d08385ea0cfbcf5c7b308701d810a1bd19036",
	)

	assertContainerId(t, `
11:devices:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
10:freezer:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
9:blkio:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
8:pids:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
7:cpuset:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
6:hugetlb:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
5:perf_event:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
4:memory:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
3:cpu,cpuacct:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
2:net_cls,net_prio:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
1:name=systemd:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod9a43c98c_c0dd_4642_bf7d_acdb53d443ec.slice/cri-containerd-31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567.scope
`,
		"31df9b0d9819f8b4c80f502880afaa4bd71079514cd0a90b0b47db972fdc9567",
	)

	assertContainerId(t, `
12:devices:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
11:blkio:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
10:perf_event:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
9:rdma:/
8:memory:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
7:net_cls,net_prio:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
6:cpuset:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
5:hugetlb:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
4:pids:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
3:freezer:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
2:cpu,cpuacct:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
1:name=systemd:/kubepods/burstable/poda0178de9-9963-4c21-a277-14564e7abd2b/b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69
0::/system.slice/containerd.service
`,
		"b08be0f4a5536647b2ec2c5c8bd03d928ca897b483528848e716ce80ed37ec69",
	)

	assertContainerId(t, "0::/", "")
	assertContainerId(t, "", "")
	assertContainerId(t, "/", "")
}

func TestGetAWSMetadata(t *testing.T) {
	testEc2MetadataZoneURL := "http://localhost:8880/latest/meta-data/placement/availability-zone"
	testEc2MetadataInstanceIDURL := "http://localhost:8880/latest/meta-data/instance-id"

	sm := http.NewServeMux()
	sm.HandleFunc("/latest/meta-data/instance-id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "i-12345678")
	})
	sm.HandleFunc("/latest/meta-data/placement/availability-zone", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "us-east-7")
	})

	addr := "localhost:8880"
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err)

	s := &http.Server{Addr: addr, Handler: sm}
	// change EC2 MD URLs
	go s.Serve(ln)
	defer func() { // restore old URLs
		ln.Close()
	}()
	time.Sleep(50 * time.Millisecond)

	id := getAWSMeta(testEc2MetadataInstanceIDURL)
	assert.Equal(t, "i-12345678", id)
	assert.Equal(t, "i-12345678", id)
	zone := getAWSMeta(testEc2MetadataZoneURL)
	assert.Equal(t, "us-east-7", zone)
	assert.Equal(t, "us-east-7", zone)
}

func TestGetPid(t *testing.T) {
	assert.Equal(t, os.Getpid(), getPid())
}

func TestGetHostname(t *testing.T) {
	host, _ := os.Hostname()
	assert.Equal(t, host, getHostname())
}

func TestUpdateHostId(t *testing.T) {
	lh := newLockedID()
	updateHostID(lh)
	assert.True(t, lh.ready())

	h := lh.copyID()

	host, _ := os.Hostname()
	assert.Equal(t, host, h.Hostname())
	assert.Equal(t, os.Getpid(), h.Pid())
	assert.Equal(t, getEC2ID(), h.EC2Id())
	assert.Equal(t, getEC2Zone(), h.EC2Zone())
	assert.Equal(t, getContainerID(), h.ContainerId())
	assert.Equal(t, strings.Join(getMACAddressList(), ""),
		strings.Join(h.MAC(), ""))
	assert.EqualValues(t, getHerokuDynoId(), h.HerokuId())
}

func TestUpdate(t *testing.T) {
	log.SetLevel(log.DEBUG)
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	lh := newLockedID()
	tk := make(chan struct{}, 1)
	tk <- struct{}{}

	assert.False(t, lh.ready())
	update(tk, lh)
	<-tk
	assert.True(t, lh.ready())

	for i := 0; i < 10; i++ {
		update(tk, lh)
	}
	assert.Contains(t, buf.String(), prevUpdaterRunning)
}

func returnEmpty() string { return "" }

func returnSomething() string { return "hello" }

func TestGetOrFallback(t *testing.T) {
	assert.Equal(t, "fallback",
		getOrFallback(returnEmpty, "fallback"))
	assert.Equal(t, "hello",
		getOrFallback(returnSomething, "fallback"))
}

// this line is used to set the environment variable DYNO before init runs
var _ = os.Setenv(envDyno, "test-dyno")

func TestGetHerokuDynoId(t *testing.T) {
	os.Setenv(envDyno, "test-dyno")
	assert.Equal(t, "test-dyno", getHerokuDynoId())
	os.Unsetenv(envDyno)
	assert.Equal(t, "test-dyno", getHerokuDynoId())

	os.Setenv(envDyno, "take-no-effect")
	assert.Equal(t, "test-dyno", getHerokuDynoId())
}

func TestInitDyno(t *testing.T) {
	var dynoID string
	os.Setenv(envDyno, "test-dyno")
	initDyno(&dynoID)
	assert.Equal(t, "test-dyno", dynoID)

	os.Unsetenv(envDyno)
	initDyno(&dynoID)
	assert.Equal(t, "", dynoID)
}
