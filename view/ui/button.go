package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"goengine/resource"
	"golang.org/x/image/font"
)

// Button is a clickable UI button with a label.
type Button struct {
	X, Y          int    // position in logical coordinates
	Width, Height int    // size
	Label         string // text drawn on the button
	OnClick       func() // called when the button is clicked (left mouse, just pressed)

	// Colors (optional; zero value uses defaults)
	FillColor   color.Color
	HoverColor  color.Color
	LabelColor  color.Color
}

// Default button colors.
var (
	DefaultButtonFill  = color.RGBA{60, 60, 80, 255}
	DefaultButtonHover = color.RGBA{80, 80, 120, 255}
	DefaultButtonLabel = color.White
)

// Contains reports whether the point (x, y) is inside the button.
func (b *Button) Contains(x, y int) bool {
	return x >= b.X && x < b.X+b.Width && y >= b.Y && y < b.Y+b.Height
}

// Draw renders the button on screen. face is used to draw the label.
func (b *Button) Draw(screen *ebiten.Image, face font.Face) {
	fill := b.FillColor
	if fill == nil {
		fill = DefaultButtonFill
	}
	labelClr := b.LabelColor
	if labelClr == nil {
		labelClr = DefaultButtonLabel
	}

	vector.FillRect(screen, float32(b.X), float32(b.Y), float32(b.Width), float32(b.Height), fill, false)

	if b.Label != "" && face != nil {
		img := resource.TextToImage(face, b.Label, labelClr)
		bounds := img.Bounds()
		tw := bounds.Dx()
		th := bounds.Dy()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.X+(b.Width-tw)/2), float64(b.Y+(b.Height-th)/2))
		screen.DrawImage(img, op)
	}
}

// DrawWithState is like Draw but uses hoverColor when hovered (cursor over button).
func (b *Button) DrawWithState(screen *ebiten.Image, face font.Face, hovered bool) {
	fill := b.FillColor
	if fill == nil {
		fill = DefaultButtonFill
	}
	if hovered {
		if b.HoverColor != nil {
			fill = b.HoverColor
		} else {
			fill = DefaultButtonHover
		}
	}
	labelClr := b.LabelColor
	if labelClr == nil {
		labelClr = DefaultButtonLabel
	}

	vector.FillRect(screen, float32(b.X), float32(b.Y), float32(b.Width), float32(b.Height), fill, false)

	if b.Label != "" && face != nil {
		img := resource.TextToImage(face, b.Label, labelClr)
		bounds := img.Bounds()
		tw := bounds.Dx()
		th := bounds.Dy()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.X+(b.Width-tw)/2), float64(b.Y+(b.Height-th)/2))
		screen.DrawImage(img, op)
	}
}
