// +build linux,go1.15
// +build linux,!go1.16

package auto

// CPU is a no-op on this platform.
func CPU() {}
