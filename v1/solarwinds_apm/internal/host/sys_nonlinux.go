//go:build !linux
// +build !linux

package host

// IsPhysicalInterface checks if the network interface is physical. It always
// returns true for non-Linux platforms.
func IsPhysicalInterface(ifname string) bool { return true }

// initDistro returns the ditro information of the system, it returns Unkown-not-Linux
// for non-Linux platforms.
func initDistro() string {
	return "Unknown-not-Linux"
}
