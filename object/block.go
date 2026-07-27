package object

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Block draws a solid rectangle at the GameObject's transform position.
// Useful as placeholder geometry for level blocking during early development.
type Block struct {
	Width        float64     // width in logical pixels
	Height       float64     // height in logical pixels
	Color        color.Color // fill color (nil = default gray)
	FacingMarker bool        // if true, draw a small marker strip showing which way the block faces
}

// DefaultBlockColor is used when Block.Color is nil.
var DefaultBlockColor = color.RGBA{100, 100, 120, 255}

// FacingMarkerColor is the marker strip drawn when Block.FacingMarker is true.
var FacingMarkerColor = color.RGBA{255, 230, 60, 255}

// Type implements Component.
func (Block) Type() string { return "block" }

// Clone implements Component.
func (b *Block) Clone() Component {
	clone := *b
	return &clone
}

// Draw implements Drawer. Draws a filled rect at the transform's position.
// When FacingMarker is set, a narrow strip is also drawn on the block's right edge, mirrored to
// the left edge when transform.ScaleX < 0 -- the same left/right flip convention Sprite and
// Spritesheet use for facing direction (see their GeoM.Scale(transform.ScaleX, ...) calls), so a
// script can flip a Block's facing marker with the same self.set_facing() call that flips a sprite.
func (b *Block) Draw(screen *ebiten.Image, transform *Transform) {
	if transform == nil || b.Width <= 0 || b.Height <= 0 {
		return
	}
	clr := b.Color
	if clr == nil {
		clr = DefaultBlockColor
	}
	vector.FillRect(screen, float32(transform.X), float32(transform.Y), float32(b.Width), float32(b.Height), clr, false)

	if b.FacingMarker {
		markerWidth := b.Width * 0.25
		markerX := transform.X + b.Width - markerWidth // default: facing right
		if transform.ScaleX < 0 {
			markerX = transform.X // facing left
		}
		vector.FillRect(screen, float32(markerX), float32(transform.Y), float32(markerWidth), float32(b.Height), FacingMarkerColor, false)
	}
}
