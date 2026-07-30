//go:build !linux && !darwin

package scene

// cpuPercent has no portable implementation outside linux/darwin (syscall.Getrusage's Rusage/
// Timeval shape is POSIX-specific, and this covers both Windows and the js/wasm build) -- see
// debug_stats_posix.go for the real implementation.
func cpuPercent() (float64, bool) {
	return 0, false
}

// peakRSSBytes has no portable implementation outside linux/darwin -- see cpuPercent above and
// debug_stats_posix.go's peakRSSBytes for the real implementation.
func peakRSSBytes() (int64, bool) {
	return 0, false
}
