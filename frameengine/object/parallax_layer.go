package object

import "github.com/hajimehoshi/ebiten/v2"

// ParallaxLayer marks a GameObject as a scrolling background layer. Unlike Sprite/Spritesheet, it
// does not implement Drawer -- it carries no camera information, and parallax scroll is inherently
// camera-relative -- so it is never drawn by the generic Manager.Draw path. Instead
// scene.WorldScene draws it directly onto the screen (see scene/parallax.go), offset by the
// camera's scroll scaled by ScrollFactor, before the level-sized world buffer everything else
// draws into.
type ParallaxLayer struct {
	Image *ebiten.Image

	// ScrollFactor is this layer's speed relative to the camera: 0 keeps it fixed to the screen
	// (reads as infinitely distant, e.g. a sky), 1 scrolls it 1:1 with gameplay (the same rate as
	// everything drawn into the world buffer), and values in between (e.g. 0.3) sit further back,
	// producing the classic multi-layer depth illusion.
	ScrollFactor float64

	// Repeat tiles Image horizontally to cover the full viewport width, for source art that is a
	// narrow strip meant to repeat (e.g. a corridor wall panel or a floor tile) rather than a
	// single image meant to be stretched or shown once.
	Repeat bool
}

// Type implements Component.
func (ParallaxLayer) Type() string { return "parallax_layer" }

// Clone implements Component. The clone shares the same Image (a cached, read-only asset), not a copy of it.
func (p *ParallaxLayer) Clone() Component {
	clone := *p
	return &clone
}
