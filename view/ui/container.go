package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
)

// Container holds UI elements (e.g. buttons) and handles input and drawing.
// It converts cursor to logical coordinates and tracks input internally.
type Container struct {
	Buttons  []*Button
	cursorX  int
	cursorY  int
}

// NewContainer returns a new empty container.
func NewContainer() *Container {
	return &Container{
		Buttons: nil,
	}
}

// AddButton adds a button to the container.
func (c *Container) AddButton(b *Button) {
	c.Buttons = append(c.Buttons, b)
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
	for _, b := range c.Buttons {
		if b.OnClick == nil {
			continue
		}
		if b.Contains(c.cursorX, c.cursorY) && leftJustPressed {
			b.OnClick()
		}
	}
}

// Draw renders all buttons using the cursor state from the last Update.
func (c *Container) Draw(screen *ebiten.Image, face font.Face) {
	for _, b := range c.Buttons {
		hovered := b.Contains(c.cursorX, c.cursorY)
		b.DrawWithState(screen, face, hovered)
	}
}
