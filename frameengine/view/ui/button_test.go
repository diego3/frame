package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

func newTestButton() *Button {
	return &Button{X: 10, Y: 20, Width: 100, Height: 40}
}

func TestButton_Contains(t *testing.T) {
	b := newTestButton()

	tests := []struct {
		name string
		x, y int
		want bool
	}{
		{"inside", 50, 30, true},
		{"top-left corner is inside (inclusive)", 10, 20, true},
		{"just left of the button", 9, 30, false},
		{"just above the button", 50, 19, false},
		{"right edge is exclusive", 110, 30, false},
		{"bottom edge is exclusive", 50, 60, false},
		{"far outside", 1000, 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, b.Contains(tt.x, tt.y))
		})
	}
}

func TestButton_HandleClick_nilOnClickDoesNotPanic(t *testing.T) {
	b := newTestButton()
	assert.NotPanics(t, b.HandleClick)
}

func TestButton_HandleClick_invokesOnClick(t *testing.T) {
	b := newTestButton()
	calls := 0
	b.OnClick = func() { calls++ }

	b.HandleClick()
	b.HandleClick()

	assert.Equal(t, 2, calls)
}

func TestButton_Draw_doesNotPanic(t *testing.T) {
	screen := ebiten.NewImage(200, 200)

	tests := []struct {
		name    string
		button  *Button
		hovered bool
	}{
		{"defaults, not hovered", newTestButton(), false},
		{"defaults, hovered", newTestButton(), true},
		{"with label but no face", &Button{X: 0, Y: 0, Width: 50, Height: 20, Label: "Click me"}, false},
		{"with custom colors", &Button{
			X: 0, Y: 0, Width: 50, Height: 20,
			FillColor: nil, HoverColor: nil, LabelColor: nil,
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { tt.button.Draw(screen, nil, tt.hovered) })
		})
	}
}

// Button must satisfy the ui.Element interface used by Container/ports.UIRoot.
var _ Element = (*Button)(nil)
