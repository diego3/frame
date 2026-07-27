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
	// Scale first, then Translate: ebiten's GeoM.Scale scales the *entire* current matrix,
	// translation included, so Translate-then-Scale would also scale the position (e.g. a
	// character at X=100 with ScaleX=2 would be drawn at X=200) instead of just resizing/
	// flipping the image in place before it's moved to its world position.
	op.GeoM.Scale(transform.ScaleX, transform.ScaleY)
	// A negative scale flips the image about its own local origin (0,0), leaving it spanning
	// [-w*|scale|, 0] instead of [0, w*|scale|] -- left unadjusted, the drawn footprint would jump
	// to the opposite side of transform.X/Y depending on facing. Shifting by the scaled image size
	// keeps the footprint anchored at transform.X/Y consistently regardless of sign, matching how
	// Block already behaves (see object.Block.Draw).
	b := s.Image.Bounds()
	tx, ty := transform.X, transform.Y
	if transform.ScaleX < 0 {
		tx += float64(b.Dx()) * -transform.ScaleX
	}
	if transform.ScaleY < 0 {
		ty += float64(b.Dy()) * -transform.ScaleY
	}
	op.GeoM.Translate(tx, ty)
	screen.DrawImage(s.Image, op)
}
