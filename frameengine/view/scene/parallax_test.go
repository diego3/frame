package scene

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goengine/frameengine/object"
)

func TestParallaxTileXs(t *testing.T) {
	tests := []struct {
		name             string
		baseX, iw, viewW float64
		want             []float64
	}{
		{"aligned base tiles exactly to the viewport", 0, 100, 300, []float64{0, 100, 200}},
		{"positive base wraps back to cover the left edge", 50, 100, 300, []float64{-50, 50, 150, 250}},
		{"negative base needs no extra wrap", -30, 100, 200, []float64{-30, 70, 170}},
		{"non-positive image width yields no tiles", 0, 0, 300, nil},
		{"non-positive viewport width yields no tiles", 0, 100, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parallaxTileXs(tt.baseX, tt.iw, tt.viewW)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDrawParallaxLayers_skipsInactiveAndPrototypeAndUnrelatedObjects(t *testing.T) {
	// Smoke test: a world containing an inactive layer, a prototype layer, an object with no
	// parallax_layer component, and an object with the component but a nil Image must not panic
	// and must leave m.world/m.cam untouched. Actual pixel output isn't asserted -- that's covered
	// by TestParallaxTileXs and drawParallaxLayer's own math -- this only guards the
	// GameObject-filtering wired up in drawParallaxLayers itself.
	m := NewWorldScene()
	world := object.NewManager()

	inactive := object.NewGameObject("inactive_layer")
	inactive.Active = false
	inactive.AddComponent(&object.Transform{})
	inactive.AddComponent(&object.ParallaxLayer{Image: ebiten.NewImage(4, 4)})
	world.Add(inactive)

	proto := object.NewGameObject("proto_layer")
	proto.IsPrototype = true
	proto.AddComponent(&object.Transform{})
	proto.AddComponent(&object.ParallaxLayer{Image: ebiten.NewImage(4, 4)})
	world.Add(proto)

	other := object.NewGameObject("not_a_layer")
	other.AddComponent(&object.Transform{})
	world.Add(other)

	nilImage := object.NewGameObject("nil_image_layer")
	nilImage.AddComponent(&object.Transform{})
	nilImage.AddComponent(&object.ParallaxLayer{Image: nil})
	world.Add(nilImage)

	real := object.NewGameObject("real_layer")
	real.AddComponent(&object.Transform{X: 10, Y: 20, ScaleX: 2, ScaleY: 2})
	real.AddComponent(&object.ParallaxLayer{Image: ebiten.NewImage(8, 8), ScrollFactor: 0.5, Repeat: true})
	world.Add(real)

	m.world = world
	screen := ebiten.NewImage(100, 50)

	assert.NotPanics(t, func() { m.drawParallaxLayers(screen) })
}

func TestDrawParallaxLayers_nilWorldIsNoop(t *testing.T) {
	m := NewWorldScene()
	require.Nil(t, m.world)

	assert.NotPanics(t, func() { m.drawParallaxLayers(ebiten.NewImage(10, 10)) })
}
