package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font"
)

// fakeElement records how Container.Draw invokes it, without depending on ebiten's real input
// state (Container.Update reads live cursor/mouse state, which isn't available outside a running
// game loop, so these tests exercise element management and Draw only).
type fakeElement struct {
	containsResult bool
	drawCalls      int
	lastHovered    bool
}

func (f *fakeElement) Contains(x, y int) bool { return f.containsResult }
func (f *fakeElement) HandleClick()           {}
func (f *fakeElement) Draw(screen *ebiten.Image, face font.Face, hovered bool) {
	f.drawCalls++
	f.lastHovered = hovered
}

func TestContainer_NewContainer_isEmpty(t *testing.T) {
	c := NewContainer()
	assert.Empty(t, c.Elements)
}

func TestContainer_AddElement(t *testing.T) {
	c := NewContainer()
	e1 := &fakeElement{}
	e2 := &fakeElement{}

	c.AddElement(e1)
	c.AddElement(e2)

	require.Len(t, c.Elements, 2)
	assert.Same(t, e1, c.Elements[0])
	assert.Same(t, e2, c.Elements[1])
}

func TestContainer_AddButton_wrapsAsElement(t *testing.T) {
	c := NewContainer()
	b := newTestButton()

	c.AddButton(b)

	require.Len(t, c.Elements, 1)
	assert.Same(t, Element(b), c.Elements[0])
}

func TestContainer_Draw_drawsEveryElementWithHoverState(t *testing.T) {
	c := NewContainer()
	hovered := &fakeElement{containsResult: true}
	notHovered := &fakeElement{containsResult: false}
	c.AddElement(hovered)
	c.AddElement(notHovered)

	screen := ebiten.NewImage(10, 10)
	c.Draw(screen, nil)

	assert.Equal(t, 1, hovered.drawCalls)
	assert.True(t, hovered.lastHovered)
	assert.Equal(t, 1, notHovered.drawCalls)
	assert.False(t, notHovered.lastHovered)
}
