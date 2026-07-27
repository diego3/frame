package resource

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// TextToImage renders text with the given face and color to a new *ebiten.Image. Useful for drawing labels or UI.
func TextToImage(face font.Face, text string, clr color.Color) *ebiten.Image {
	dr := &font.Drawer{
		Face: face,
	}
	adv := dr.MeasureString(text)
	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Ceil()
	width := adv.Ceil()
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	dr.Dst = img
	dr.Src = image.NewUniform(clr)
	dr.Dot = fixed.P(0, metrics.Ascent.Ceil())
	dr.DrawString(text)
	return ebiten.NewImageFromImage(img)
}
