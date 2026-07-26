package object

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Sprite draws an image at the GameObject's transform position.
type Sprite struct {
	Image *ebiten.Image
	Layer int // draw order (higher = on top)
}

// Type implements Component.
func (Sprite) Type() string { return "sprite" }

// Clone implements Component. The clone shares the same Image (a cached, read-only asset), not a copy of it.
func (s *Sprite) Clone() Component {
	clone := *s
	return &clone
}

// Draw implements Drawer. Draws the image at the transform's position (top-left) with scale applied.
func (s *Sprite) Draw(screen *ebiten.Image, transform *Transform) {
	if s.Image == nil || transform == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(transform.X, transform.Y)
	op.GeoM.Scale(transform.ScaleX, transform.ScaleY)
	screen.DrawImage(s.Image, op)
}
