//go:build !unix

package load

// processCPUSeconds has no portable implementation off unix; observed CPU is
// simply omitted from the final log record there.
func processCPUSeconds() (float64, bool) { return 0, false }
