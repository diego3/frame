package object

import "github.com/hajimehoshi/ebiten/v2"

// Component is the minimal interface for something attached to a GameObject.
// Type() must return a unique name for the component kind (e.g. "transform", "sprite").
type Component interface {
	Type() string
}

// Updater is implemented by components that run logic each frame.
type Updater interface {
	Component
	Update(dt float64)
}

// Drawer is implemented by components that draw. Receives the screen and the GameObject's transform for position.
type Drawer interface {
	Component
	Draw(screen *ebiten.Image, transform *Transform)
}
