// © 2024 SolarWinds Worldwide, LLC. All rights reserved.
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

package metrics

import (
	"context"
	"runtime"

	"github.com/solarwinds/apm-go/internal/host"
	"go.opentelemetry.io/otel/metric"
)

func RegisterOtelRuntimeMetrics(mp metric.MeterProvider) error {
	meter := mp.Meter("sw.apm.runtime.metrics")
	numGoroutine, err := meter.Int64ObservableGauge("trace.go.runtime.NumGoroutine")
	if err != nil {
		return err
	}
	numCgoCall, err := meter.Int64ObservableGauge("trace.go.runtime.NumCgoCall")
	if err != nil {
		return err
	}

	lastGC, err := meter.Int64ObservableGauge("trace.go.gc.LastGC")
	if err != nil {
		return err
	}
	nextGC, err := meter.Int64ObservableGauge("trace.go.gc.NextGC")
	if err != nil {
		return err
	}
	pauseTotalNs, err := meter.Int64ObservableGauge("trace.go.gc.PauseTotalNs")
	if err != nil {
		return err
	}
	numGC, err := meter.Int64ObservableGauge("trace.go.gc.NumGC")
	if err != nil {
		return err
	}
	numForcedGC, err := meter.Int64ObservableGauge("trace.go.gc.NumForcedGC")
	if err != nil {
		return err
	}
	GCCPUFraction, err := meter.Float64ObservableGauge("trace.go.gc.GCCPUFraction")
	if err != nil {
		return err
	}

	alloc, err := meter.Int64ObservableGauge("trace.go.memory.Alloc")
	if err != nil {
		return err
	}
	totalAlloc, err := meter.Int64ObservableGauge("trace.go.memory.TotalAlloc")
	if err != nil {
		return err
	}
	sys, err := meter.Int64ObservableGauge("trace.go.memory.Sys")
	if err != nil {
		return err
	}
	lookups, err := meter.Int64ObservableGauge("trace.go.memory.Lookups")
	if err != nil {
		return err
	}
	mallocs, err := meter.Int64ObservableGauge("trace.go.memory.Mallocs")
	if err != nil {
		return err
	}
	frees, err := meter.Int64ObservableGauge("trace.go.memory.Frees")
	if err != nil {
		return err
	}
	heapAlloc, err := meter.Int64ObservableGauge("trace.go.memory.HeapAlloc")
	if err != nil {
		return err
	}
	heapSys, err := meter.Int64ObservableGauge("trace.go.memory.HeapSys")
	if err != nil {
		return err
	}
	heapIdle, err := meter.Int64ObservableGauge("trace.go.memory.HeapIdle")
	if err != nil {
		return err
	}
	heapInuse, err := meter.Int64ObservableGauge("trace.go.memory.HeapInuse")
	if err != nil {
		return err
	}
	heapReleased, err := meter.Int64ObservableGauge("trace.go.memory.HeapReleased")
	if err != nil {
		return err
	}
	heapObjects, err := meter.Int64ObservableGauge("trace.go.memory.HeapObjects")
	if err != nil {
		return err
	}
	stackInuse, err := meter.Int64ObservableGauge("trace.go.memory.StackInuse")
	if err != nil {
		return err
	}
	stackSys, err := meter.Int64ObservableGauge("trace.go.memory.StackSys")
	if err != nil {
		return err
	}

	hostMemoryTotalRAM, err := meter.Int64ObservableGauge("trace.go.memory.TotalRAM")
	if err != nil {
		return err
	}
	hostMemoryfreeRAM, err := meter.Int64ObservableGauge("trace.go.memory.FreeRAM")
	if err != nil {
		return err
	}
	hostSystemLoad1, err := meter.Float64ObservableGauge("trace.go.system.Load1")
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(
		func(_ context.Context, obs metric.Observer) error {
			// category runtime
			obs.ObserveInt64(numGoroutine, int64(runtime.NumGoroutine()))
			obs.ObserveInt64(numCgoCall, int64(runtime.NumCgoCall()))

			var mem runtime.MemStats
			host.Mem(&mem)
			// category gc
			obs.ObserveInt64(lastGC, int64(mem.LastGC))
			obs.ObserveInt64(nextGC, int64(mem.NextGC))
			obs.ObserveInt64(pauseTotalNs, int64(mem.PauseTotalNs))
			obs.ObserveInt64(numGC, int64(mem.NumGC))
			obs.ObserveInt64(numForcedGC, int64(mem.NumForcedGC))
			obs.ObserveFloat64(GCCPUFraction, mem.GCCPUFraction)

			// category memory
			obs.ObserveInt64(alloc, int64(mem.Alloc))
			obs.ObserveInt64(totalAlloc, int64(mem.TotalAlloc))
			obs.ObserveInt64(sys, int64(mem.Sys))
			obs.ObserveInt64(lookups, int64(mem.Lookups))
			obs.ObserveInt64(mallocs, int64(mem.Mallocs))
			obs.ObserveInt64(frees, int64(mem.Frees))
			obs.ObserveInt64(heapAlloc, int64(mem.HeapAlloc))
			obs.ObserveInt64(heapSys, int64(mem.HeapSys))
			obs.ObserveInt64(heapIdle, int64(mem.HeapIdle))
			obs.ObserveInt64(heapInuse, int64(mem.HeapInuse))
			obs.ObserveInt64(heapReleased, int64(mem.HeapReleased))
			obs.ObserveInt64(heapObjects, int64(mem.HeapObjects))
			obs.ObserveInt64(stackInuse, int64(mem.StackInuse))
			obs.ObserveInt64(stackSys, int64(mem.StackSys))

			if hostMetrics, ok := getHostMetrics(); ok {
				obs.ObserveInt64(hostMemoryTotalRAM, int64(hostMetrics.memoryTotalRAM))
				obs.ObserveInt64(hostMemoryfreeRAM, int64(hostMetrics.memoryfreeRAM))
				obs.ObserveFloat64(hostSystemLoad1, hostMetrics.systemLoad1)
			}

			return nil
		},
		numGoroutine, numCgoCall, lastGC, nextGC, pauseTotalNs, numGC,
		numForcedGC, GCCPUFraction, alloc, totalAlloc, sys, lookups,
		mallocs, frees, heapAlloc, heapSys, heapIdle, heapInuse,
		heapReleased, heapObjects, stackInuse, stackSys,
	)
	return err
}
