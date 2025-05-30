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
	"reflect"
	"sync"

	"github.com/solarwinds/apm-go/internal/instance"
	"github.com/solarwinds/apm-go/internal/log"
)

// logging texts
const (
	partialUpdateNotAllowed = "Partial Update is not allowed: %d."
	hostIdInitDone          = "HostID initialization done."
)

// caches and their sync.Once protectors
var (
	containerId     string
	containerIdOnce sync.Once

	// the Heroku DYNO id
	dyno     string
	dynoOnce sync.Once

	// the Azure web application instance ID
	azureAppInstId     string
	azureAppInstIdOnce sync.Once
)

// lockedID is a ID protected by a mutex. To avoid being modified without
// mutex protected, the caller can only get a copyID of the internal ID.
type lockedID struct {
	sync.RWMutex

	// it will be closed when the ID is initialized for the first time
	c chan struct{}

	// make sure channel c is not close twice
	cClosed *sync.Once

	id *ID
}

// newLockedID returns an uninitialized lockedID, it can be used only after
// the first full update (see fullUpdate())
func newLockedID() *lockedID {
	return &lockedID{
		RWMutex: sync.RWMutex{},
		c:       make(chan struct{}),
		cClosed: &sync.Once{},
		id:      newID(),
	}
}

func (h *ID) copy() *ID {
	c := newID()
	c.update(
		withHostname(h.hostname),
		withPid(h.pid), // pid doesn't change, but we fullUpdate it anyways
		withContainerId(h.containerId),
		withMAC(h.mac),
		withHerokuId(h.herokuId),
		withAzureAppInstId(h.azureAppInstId),
	)
	return c
}

// fullUpdate update all the fields of HostID with the setters. Unlike HostID's
// update function, partial update is not allowed here.
func (lh *lockedID) fullUpdate(setters ...IDSetter) {
	lh.Lock()
	defer lh.Unlock()

	if len(setters) != reflect.ValueOf(lh.id).Elem().NumField() {
		log.Debugf(partialUpdateNotAllowed, len(setters))
		return
	}

	lh.id.update(setters...)
	lh.setReady()
}

// setReady changes the status of the lockID to 'ready'.
func (lh *lockedID) setReady() {
	select {
	case <-lh.c:
		return
	default:
		lh.cClosed.Do(func() {
			close(lh.c)
			log.Info(hostIdInitDone)
		})
	}
}

// ready returns if the lockedID is initialized.
func (lh *lockedID) ready() bool {
	select {
	case <-lh.c:
		return true
	default:
		return false
	}
}

// waitForReady blocks until the lockedID is ready (initialized)
func (lh *lockedID) waitForReady() {
	<-lh.c
}

// copyID returns a copy of the lockedID's internal HostID. However, it doesn't
// check if the internal HostID has been initialized.
func (lh *lockedID) copyID() *ID {
	lh.RLock()
	defer lh.RUnlock()

	return lh.id.copy()
}

// ID defines the minimum set of host metadata that identifies a host
type ID struct {
	// the hostname of this host
	hostname string

	// process ID
	pid int

	// container ID
	containerId string

	// the list of MAC addresses
	mac []string

	// The Heroku DynoID
	herokuId string

	// The Azure's WEBAPP_INSTANCE_ID
	azureAppInstId string
}

// Hostname returns the hostname field of ID
func (h *ID) Hostname() string {
	return h.hostname
}

// Pid returns the pid field of ID
func (h *ID) Pid() int {
	return h.pid
}

// ContainerId returns the containerId field of ID
func (h *ID) ContainerId() string {
	return h.containerId
}

// MAC returns the mac field of ID
func (h *ID) MAC() []string {
	return h.mac
}

// HerokuId returns the herokuId field of ID
func (h *ID) HerokuId() string {
	return h.herokuId
}

// AzureAppInstId returns the Azure's web application instance ID
func (h *ID) AzureAppInstId() string {
	return h.azureAppInstId
}

// InstanceID returns a string of a version 4 UUID, generated on startup
func (h *ID) InstanceID() string {
	return instance.InstanceID()
}

// IDSetter defines a function type which set a field of ID
type IDSetter func(h *ID)

func withHostname(hostname string) IDSetter {
	return func(h *ID) {
		h.hostname = hostname
	}
}

func withPid(pid int) IDSetter {
	return func(h *ID) {
		h.pid = pid
	}
}

func withContainerId(id string) IDSetter {
	return func(h *ID) {
		h.containerId = id
	}
}

func withMAC(mac []string) IDSetter {
	return func(h *ID) {
		h.mac = []string{}
		h.mac = append(h.mac, mac...)
	}
}

func withHerokuId(id string) IDSetter {
	return func(h *ID) {
		h.herokuId = id
	}
}

func withAzureAppInstId(id string) IDSetter {
	return func(h *ID) {
		h.azureAppInstId = id
	}
}

func newID(setters ...IDSetter) *ID {
	h := &ID{}
	h.update(setters...)
	return h
}

func (h *ID) update(setters ...IDSetter) {
	for _, fn := range setters {
		fn(h)
	}
}
