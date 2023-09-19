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
	"bufio"
	"github.com/solarwinds/apm-go/internal/log"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	// the interval to update the metadata periodically
	observeInterval = time.Minute

	// the environment variable for Heroku DYNO ID
	envDyno = "DYNO"

	// the environment variable for Azure's WEBAPP_INSTANCE_ID
	envAzureAppInstId = "WEBSITE_INSTANCE_ID"
)

// logging texts
const (
	hostObserverStarted = "Host metadata observer started."
	hostObserverStopped = "Host metadata observer stopped."
	prevUpdaterRunning  = "The previous updater is still running."
)

// observer checks the update of the host metadata periodically. It runs in a
// standalone goroutine.
func observer() {
	log.Debug(hostObserverStarted)
	defer log.Info(hostObserverStopped)

	// Only one hostID updater is allowed at a time.
	token := make(chan struct{}, 1)
	token <- struct{}{}

	// initialize the hostID as soon as possible
	update(token, hostId)

	roundup := time.Now().Truncate(observeInterval).Add(observeInterval)

	// double check before sleeping, so it won't be sleeping uselessly if
	// the agent is rejected by the collector immediately after start.
	select {
	case <-exit:
		return
	default:
	}

	// Sleep returns immediately if roundup is before time.Now()
	time.Sleep(time.Until(roundup))

	tk := time.NewTicker(observeInterval)
	defer func() { tk.Stop() }()

loop:
	for {
		update(token, hostId)

		select {
		case <-tk.C:

		case <-exit:
			break loop
		}
	}
}

// getOrFallback runs the function provided, and returns the fallback value if
// the function executed returns an empty string
func getOrFallback(fn func() string, fb string) string {
	if s := fn(); s != "" {
		return s
	}
	return fb
}

// update does the host metadata update work. The number of concurrent
// updaters are constrained by the number of elements in the token channel.
func update(token chan struct{}, lh *lockedID) {
	select {
	case <-token:
		go func(lh *lockedID) {
			updateHostID(lh)
			token <- struct{}{}
		}(lh)
	default:
		log.Debug(prevUpdaterRunning)
	}
}

func updateHostID(lh *lockedID) {
	old := lh.copyID()

	// compare and fallback if error happens
	hostname := getOrFallback(getHostname, old.hostname)
	pid := PID()
	cid := getOrFallback(getContainerID, old.containerId)
	herokuId := getOrFallback(getHerokuDynoId, old.herokuId)
	azureId := getOrFallback(getAzureAppInstId, old.azureAppInstId)

	mac := getMACAddressList()
	if len(mac) == 0 {
		mac = old.mac
	}

	setters := []IDSetter{
		withHostname(hostname),
		withPid(pid),
		withContainerId(cid),
		withMAC(mac),
		withHerokuId(herokuId),
		withAzureAppInstId(azureId),
	}

	lh.fullUpdate(setters...)
}

// getHostname is the implementation of getting the hostname
func getHostname() string {
	h, err := os.Hostname()
	if err == nil {
		hm.Lock()
		hostname = h
		hm.Unlock()
	}
	return h
}

func getPid() int {
	return os.Getpid()
}

// getContainerID fetches the container ID by reading '/proc/self/cgroup'
func getContainerID() (id string) {
	containerIdOnce.Do(func() {
		containerId = getContainerIdFromFile("/proc/self/cgroup")
		log.Debugf("Got and cached container id: %s", containerId)
	})

	return containerId
}

var containerIdRegex = regexp.MustCompile(`\A[a-f0-9]{64}\z`)

func getContainerIdFromFile(fn string) string {
	if f, err := os.Open(fn); err != nil {
		log.Debugf("failed to open cgroup file: %s", err)
	} else {
		defer func() {
			if err = f.Close(); err != nil {
				log.Debugf("failed to close cgroup file %s", err)
			}
		}()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if cid := findContainerId(scanner.Text()); cid != "" {
				return cid
			}
		}
	}
	return ""
}

func findContainerId(line string) string {
	if len(line) < 64 {
		return ""
	}
	lastSlashIdx := strings.LastIndex(line, "/")
	if lastSlashIdx == -1 {
		return ""
	}
	var containerId string
	lastSection := line[lastSlashIdx+1:]
	colonIdx := strings.LastIndex(lastSection, ":")
	if colonIdx > -1 {
		// since containerd v1.5.0+, containerId is divided by the last colon when the cgroupDriver is systemd:
		// https://github.com/containerd/containerd/blob/release/1.5/pkg/cri/server/helpers_linux.go#L64
		containerId = lastSection[colonIdx+1:]
	} else {
		startIdx := strings.LastIndex(lastSection, "-")
		if startIdx == -1 {
			startIdx = 0
		} else {
			startIdx++
		}

		endIdx := strings.LastIndex(lastSection, ".")
		if endIdx == -1 {
			endIdx = len(lastSection)
		}
		if startIdx > endIdx {
			return ""
		}
		containerId = lastSection[startIdx:endIdx]
	}

	if containerIdRegex.MatchString(containerId) {
		return containerId
	}
	return ""
}

// gets a comma-separated list of MAC addresses
func getMACAddressList() []string {
	var macAddrs []string

	if ifaces, err := FilteredIfaces(); err != nil {
		return macAddrs
	} else {
		for _, iface := range ifaces {
			if mac := iface.HardwareAddr.String(); mac != "" {
				macAddrs = append(macAddrs, iface.HardwareAddr.String())
			}
		}
	}

	return macAddrs
}

func getHerokuDynoId() string {
	dynoOnce.Do(func() {
		initDyno(&dyno)
	})
	return dyno
}

func getAzureAppInstId() string {
	azureAppInstIdOnce.Do(func() {
		initAzureAppInstId(&azureAppInstId)
		log.Debugf("Got and cached Azure webapp instance id: %s", azureAppInstId)
	})
	return azureAppInstId
}

func initDyno(dyno *string) {
	if d, has := os.LookupEnv(envDyno); has {
		*dyno = d
	} else {
		*dyno = ""
	}
}

func initAzureAppInstId(azureId *string) {
	if a, has := os.LookupEnv(envAzureAppInstId); has {
		*azureId = a
	} else {
		*azureId = ""
	}
}
