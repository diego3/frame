//go:build linux || darwin

package scene

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPUPercent(t *testing.T) {
	pct, ok := cpuPercent()
	require.True(t, ok, "Getrusage should succeed on linux/darwin")
	assert.GreaterOrEqual(t, pct, 0.0)

	// Within the 1s sampling window, a second call must return the exact same cached value rather
	// than resampling.
	pct2, ok2 := cpuPercent()
	require.True(t, ok2)
	assert.Equal(t, pct, pct2, "calls inside the 1s window should return the cached sample")
}

func TestPeakRSSBytes(t *testing.T) {
	rss, ok := peakRSSBytes()
	require.True(t, ok, "Getrusage should succeed on linux/darwin")
	// A running test binary has allocated at least a few hundred KB by this point regardless of
	// platform/unit normalization -- a loose lower bound that would still catch a units bug (e.g.
	// forgetting the Linux Maxrss-is-KB conversion, which would under-report by 1024x).
	assert.Greater(t, rss, int64(100*1024))
}
