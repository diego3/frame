package object

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Block draws a solid rectangle at the GameObject's transform position.
// Useful as placeholder geometry for level blocking during early development.
type Block struct {
	Width  float64     // width in logical pixels
	Height float64     // height in logical pixels
	Color  color.Color // fill color (nil = default gray)
}

// DefaultBlockColor is used when Block.Color is nil.
var DefaultBlockColor = color.RGBA{100, 100, 120, 255}

// Type implements Component.
func (Block) Type() string { return "block" }

// Draw implements Drawer. Draws a filled rect at the transform's position.
func (b *Block) Draw(screen *ebiten.Image, transform *Transform) {
	if transform == nil || b.Width <= 0 || b.Height <= 0 {
		return
	}
	clr := b.Color
	if clr == nil {
		clr = DefaultBlockColor
	}
	vector.FillRect(screen, float32(transform.X), float32(transform.Y), float32(b.Width), float32(b.Height), clr, false)
}
