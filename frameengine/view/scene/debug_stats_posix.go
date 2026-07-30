//go:build linux || darwin

package scene

import (
	"runtime"
	"sync"
	"syscall"
	"time"
)

var rusageSample struct {
	mu         sync.Mutex
	wall       time.Time
	cpu        time.Duration
	cpuPercent float64
	maxRSS     int64 // raw Rusage.Maxrss, unit differs by GOOS -- see peakRSSBytes
	ok         bool  // false until the first successful Getrusage
}

// sampleRusage refreshes the shared Getrusage-derived sample at most once per second and caches it
// in between -- Getrusage itself is cheap, but computing a rate from a sub-16ms window (i.e. every
// frame) would be both wasteful and far noisier than a ~1s window, the same reasoning FPS/TPS
// smoothing already uses. Both cpuPercent and peakRSSBytes read from this one syscall/sample so
// showing both in the debug overlay doesn't double the syscall cost.
func sampleRusage() {
	rusageSample.mu.Lock()
	defer rusageSample.mu.Unlock()

	now := time.Now()
	if !rusageSample.wall.IsZero() && now.Sub(rusageSample.wall) < time.Second {
		return
	}

	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		rusageSample.ok = false
		return
	}
	cpu := time.Duration(ru.Utime.Nano()) + time.Duration(ru.Stime.Nano())

	if rusageSample.ok && !rusageSample.wall.IsZero() {
		if wall := now.Sub(rusageSample.wall); wall > 0 {
			rusageSample.cpuPercent = float64(cpu-rusageSample.cpu) / float64(wall) * 100
		}
	}
	rusageSample.cpu = cpu
	rusageSample.maxRSS = ru.Maxrss
	rusageSample.wall = now
	rusageSample.ok = true
}

// cpuPercent returns this process' CPU usage as a percentage of one core (e.g. 150.0 means using
// roughly 1.5 cores' worth of CPU time) since the previous sample, or ok=false if Getrusage has
// never succeeded.
func cpuPercent() (float64, bool) {
	sampleRusage()
	rusageSample.mu.Lock()
	defer rusageSample.mu.Unlock()
	return rusageSample.cpuPercent, rusageSample.ok
}

// peakRSSBytes returns the process' peak resident set size in bytes (the high-water mark for real
// physical memory used, not current usage -- Go rarely returns freed heap pages to the OS, so for a
// long-running game process the two stay close after the initial warmup, but they're not the same
// thing), or ok=false if Getrusage has never succeeded. Rusage.Maxrss's unit is GOOS-dependent: KB
// on Linux, bytes on Darwin -- normalized here so callers never need to know that.
func peakRSSBytes() (int64, bool) {
	sampleRusage()
	rusageSample.mu.Lock()
	defer rusageSample.mu.Unlock()
	if !rusageSample.ok {
		return 0, false
	}
	if runtime.GOOS == "linux" {
		return rusageSample.maxRSS * 1024, true
	}
	return rusageSample.maxRSS, true
}
