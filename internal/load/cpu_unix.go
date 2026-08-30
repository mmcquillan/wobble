//go:build unix

package load

import (
	"syscall"
	"time"
)

// processCPUSeconds returns cumulative user+system CPU time for this process,
// in seconds, via getrusage(RUSAGE_SELF).
func processCPUSeconds() (float64, bool) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, false
	}
	u := time.Duration(ru.Utime.Sec)*time.Second + time.Duration(ru.Utime.Usec)*time.Microsecond
	s := time.Duration(ru.Stime.Sec)*time.Second + time.Duration(ru.Stime.Usec)*time.Microsecond
	return (u + s).Seconds(), true
}
