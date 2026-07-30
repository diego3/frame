package scene

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{"zero", 0, "0.0 MB"},
		{"exactly one megabyte", 1024 * 1024, "1.0 MB"},
		{"fractional megabytes round to one decimal", 1024*1024*3 + 512*1024, "3.5 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBytes(tt.n))
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"shorter than max is unchanged", "OpenGL 4.6", 32, "OpenGL 4.6"},
		{"exactly max is unchanged", "12345", 5, "12345"},
		{"longer than max is cut with ellipsis", "OpenGL 4.6 (Compat Profile) Mesa 23.2.1", 10, "OpenGL ..."},
		{"max at or below ellipsis width truncates with no ellipsis", "abcdef", 3, "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncate(tt.s, tt.max))
		})
	}
}

func TestDrawDebugStats_doesNotPanic(t *testing.T) {
	screen := ebiten.NewImage(960, 540)

	assert.NotPanics(t, func() { drawDebugStats(screen, 42) })
}
