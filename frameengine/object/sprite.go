package object

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Sprite draws an image at the GameObject's transform position.
type Sprite struct {
	Image *ebiten.Image
	Layer int // draw order (higher = on top)

	// RepeatWidth, when > 0, tiles Image horizontally (starting at the transform's position,
	// scaled the same as a single draw would be) until that total world-space width is covered,
	// instead of drawing Image once -- for level art that is a small repeating strip (e.g. a
	// ground or platform-top texture) rather than a single image meant to be stretched. Unlike
	// ParallaxLayer.Repeat, this needs no camera position: it tiles in world space, drawn once per
	// frame through the normal Drawer path like any other GameObject.
	RepeatWidth float64
}

// Type implements Component.
func (Sprite) Type() string { return "sprite" }

// Clone implements Component. The clone shares the same Image (a cached, read-only asset), not a copy of it.
func (s *Sprite) Clone() Component {
	clone := *s
	return &clone
}

// Draw implements Drawer. Draws the image at the transform's position (top-left) with scale
// applied, or -- when RepeatWidth > 0 -- tiles it left to right (see RepeatWidth's doc comment)
// starting at that same position to cover RepeatWidth of world space.
func (s *Sprite) Draw(screen *ebiten.Image, transform *Transform) {
	if s.Image == nil || transform == nil {
		return
	}
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
	// RepeatWidth tiling only makes sense advancing in +x from tx with a positive scaled tile
	// width; a non-positive ScaleX (unscaled-away or already flip-adjusted above) falls back to
	// drawing once, same as RepeatWidth <= 0.
	scaledWidth := float64(b.Dx()) * transform.ScaleX
	for _, x := range spriteRepeatXs(tx, scaledWidth, s.RepeatWidth) {
		drawSpriteImage(screen, s.Image, x, ty, transform.ScaleX, transform.ScaleY)
	}
}

// spriteRepeatXs returns the x positions at which a RepeatWidth Sprite's (already scaled) image
// must be drawn, starting at tx, to cover repeatWidth of world space -- or just []float64{tx} when
// repeatWidth/scaledWidth isn't a usable tiling request (RepeatWidth unset, or a non-positive
// scaled width, e.g. from zero/negative ScaleX), matching a plain single-draw Sprite. Pulled out as
// a pure function so the tiling math is unit-testable without an *ebiten.Image, same as
// scene.parallaxTileXs.
func spriteRepeatXs(tx, scaledWidth, repeatWidth float64) []float64 {
	if repeatWidth <= 0 || scaledWidth <= 0 {
		return []float64{tx}
	}
	var xs []float64
	for x := tx; x < tx+repeatWidth; x += scaledWidth {
		xs = append(xs, x)
	}
	return xs
}

func drawSpriteImage(screen, img *ebiten.Image, x, y, scaleX, scaleY float64) {
	op := &ebiten.DrawImageOptions{}
	// Scale first, then Translate: ebiten's GeoM.Scale scales the *entire* current matrix,
	// translation included, so Translate-then-Scale would also scale the position (e.g. a
	// character at X=100 with ScaleX=2 would be drawn at X=200) instead of just resizing/
	// flipping the image in place before it's moved to its world position.
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}
