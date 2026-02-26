package object

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Ball draws a filled circle at the GameObject's transform position.
// Transform (X,Y) is the top-left of the ball's bounding box; the circle center is at (X+Radius, Y+Radius).
// Useful as placeholder for dynamic circle bodies (e.g. bouncy balls).
type Ball struct {
	Radius float64     // radius in logical pixels
	Color  color.Color // fill color (nil = default)
}

// DefaultBallColor is used when Ball.Color is nil.
var DefaultBallColor = color.RGBA{200, 80, 80, 255}

// Type implements Component.
func (Ball) Type() string { return "ball" }

// Draw implements Drawer. Draws a filled circle; center = transform + (Radius, Radius).
func (b *Ball) Draw(screen *ebiten.Image, transform *Transform) {
	if transform == nil || b.Radius <= 0 {
		return
	}
	clr := b.Color
	if clr == nil {
		clr = DefaultBallColor
	}
	cx := float32(transform.X + b.Radius)
	cy := float32(transform.Y + b.Radius)
	vector.FillCircle(screen, cx, cy, float32(b.Radius), clr, false)
}
