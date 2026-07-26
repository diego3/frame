package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
)

// Element is a UI element that a Container can hit-test, click, and draw. Button implements
// Element; future controls (labels, panels, sliders) can implement it too without changing
// ports.UIRoot or Container.
type Element interface {
	// Contains reports whether the point (x, y), in logical coordinates, is inside the element.
	Contains(x, y int) bool
	// HandleClick is called when the element is hit by a just-pressed left click.
	HandleClick()
	// Draw renders the element. hovered is true when the cursor is currently inside it.
	Draw(screen *ebiten.Image, face font.Face, hovered bool)
}
