package host

import (
	"runtime"
	"runtime/debug"
)

// Mem returns the current memory statistics. Don't call this function too often
// as it stops the world while gathering the information.
func Mem(m *runtime.MemStats) {
	runtime.ReadMemStats(m)
}

// GC collects current statistics of garbage collector
func GC(stats *debug.GCStats) {
	debug.ReadGCStats(stats)
}
