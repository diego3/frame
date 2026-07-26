package scene

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"goengine/object"
)

func TestAabbOverlap(t *testing.T) {
	tests := []struct {
		name string
		box1 [4]float64 // x, y, w, h
		box2 [4]float64
		want bool
	}{
		{"identical boxes overlap", [4]float64{0, 0, 10, 10}, [4]float64{0, 0, 10, 10}, true},
		{"partial overlap", [4]float64{0, 0, 10, 10}, [4]float64{5, 5, 10, 10}, true},
		{"touching edges do not overlap", [4]float64{0, 0, 10, 10}, [4]float64{10, 0, 10, 10}, false},
		{"separated on x", [4]float64{0, 0, 10, 10}, [4]float64{20, 0, 10, 10}, false},
		{"separated on y", [4]float64{0, 0, 10, 10}, [4]float64{0, 20, 10, 10}, false},
		{"one box fully inside the other", [4]float64{0, 0, 100, 100}, [4]float64{40, 40, 5, 5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aabbOverlap(tt.box1[0], tt.box1[1], tt.box1[2], tt.box1[3], tt.box2[0], tt.box2[1], tt.box2[2], tt.box2[3])
			assert.Equal(t, tt.want, got)
			// Overlap must be symmetric regardless of argument order.
			gotSwapped := aabbOverlap(tt.box2[0], tt.box2[1], tt.box2[2], tt.box2[3], tt.box1[0], tt.box1[1], tt.box1[2], tt.box1[3])
			assert.Equal(t, tt.want, gotSwapped)
		})
	}
}

func TestAabb(t *testing.T) {
	t.Run("no transform is not ok", func(t *testing.T) {
		go_ := object.NewGameObject("no-transform")
		_, _, _, _, ok := aabb(go_)
		assert.False(t, ok)
	})

	t.Run("uses Block size when present", func(t *testing.T) {
		go_ := object.NewGameObject("blocky")
		go_.AddComponent(&object.Transform{X: 5, Y: 6})
		go_.AddComponent(&object.Block{Width: 10, Height: 20})
		x, y, w, h, ok := aabb(go_)
		assert.True(t, ok)
		assert.Equal(t, 5.0, x)
		assert.Equal(t, 6.0, y)
		assert.Equal(t, 10.0, w)
		assert.Equal(t, 20.0, h)
	})

	t.Run("falls back to PhysicsBody size when no Block", func(t *testing.T) {
		go_ := object.NewGameObject("physics-only")
		go_.AddComponent(&object.Transform{X: 1, Y: 2})
		go_.AddComponent(&object.PhysicsBody{Width: 30, Height: 40})
		x, y, w, h, ok := aabb(go_)
		assert.True(t, ok)
		assert.Equal(t, 1.0, x)
		assert.Equal(t, 2.0, y)
		assert.Equal(t, 30.0, w)
		assert.Equal(t, 40.0, h)
	})

	t.Run("no usable size is not ok", func(t *testing.T) {
		go_ := object.NewGameObject("sizeless")
		go_.AddComponent(&object.Transform{X: 1, Y: 2})
		_, _, _, _, ok := aabb(go_)
		assert.False(t, ok)
	})
}
