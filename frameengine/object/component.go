package object

import "github.com/hajimehoshi/ebiten/v2"

// Component is the minimal interface for something attached to a GameObject.
// Type() must return a unique name for the component kind (e.g. "transform", "sprite").
// Clone() must return an independent copy — used by GameObject.Clone (see the object package's
// Prototype pattern: a template GameObject marked IsPrototype is cloned at runtime to spawn real
// instances, e.g. projectiles). A plain "copy := *c; return &copy" is correct for value-only
// components; components holding a shared read-only resource (e.g. Sprite/Spritesheet's
// *ebiten.Image) should still shallow-copy — the image is meant to be shared, not duplicated.
type Component interface {
	Type() string
	Clone() Component
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
