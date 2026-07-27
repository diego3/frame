package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
)

// Container holds UI elements (e.g. buttons) and handles input and drawing.
// It converts cursor to logical coordinates and tracks input internally.
type Container struct {
	Elements []Element
	cursorX  int
	cursorY  int
}

// NewContainer returns a new empty container.
func NewContainer() *Container {
	return &Container{
		Elements: nil,
	}
}

// AddElement adds a UI element to the container. Implements ports.UIRoot.
func (c *Container) AddElement(e Element) {
	c.Elements = append(c.Elements, e)
}

// AddButton is a convenience wrapper for adding a *Button. Kept here (not on ports.UIRoot) so the
// port stays element-agnostic; callers holding a *Container directly can still use it.
func (c *Container) AddButton(b *Button) {
	c.AddElement(b)
}

// Update processes input. Call from game Update with layout dimensions (logical size).
// Cursor position and click state are read and converted internally.
func (c *Container) Update(layoutWidth, layoutHeight int) {
	cx, cy := ebiten.CursorPosition()
	ww, wh := ebiten.WindowSize()
	if ww > 0 && wh > 0 {
		c.cursorX = cx * layoutWidth / ww
		c.cursorY = cy * layoutHeight / wh
	}
	leftJustPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	for _, e := range c.Elements {
		if e.Contains(c.cursorX, c.cursorY) && leftJustPressed {
			e.HandleClick()
		}
	}
}

// Draw renders all elements using the cursor state from the last Update.
func (c *Container) Draw(screen *ebiten.Image, face font.Face) {
	for _, e := range c.Elements {
		hovered := e.Contains(c.cursorX, c.cursorY)
		e.Draw(screen, face, hovered)
	}
}
