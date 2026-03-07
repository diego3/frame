package object

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Manager holds GameObjects and runs Update/Draw each frame.
type Manager struct {
	gameObjects []*GameObject
}

// NewManager returns an empty manager.
func NewManager() *Manager {
	return &Manager{
		gameObjects: nil,
	}
}

// Add adds a GameObject to the manager.
func (w *Manager) Add(go_ *GameObject) {
	if go_ == nil {
		return
	}
	w.gameObjects = append(w.gameObjects, go_)
}

// Find returns the first GameObject with the given name, or nil.
func (w *Manager) Find(name string) *GameObject {
	for _, go_ := range w.gameObjects {
		if go_.Name == name {
			return go_
		}
	}
	return nil
}

// Objects returns all GameObjects in the manager (read-only iteration).
func (w *Manager) Objects() []*GameObject {
	return w.gameObjects
}

// Update runs Update(dt) on Updater components of active GameObjects.
// If a GameObject has an Animator, only the spritesheet matching Animator.Current is updated (so only one animation advances).
func (w *Manager) Update(dt float64) {
	for _, go_ := range w.gameObjects {
		if !go_.Active {
			continue
		}
		anim := go_.Animator()
		if anim != nil && anim.Current != "" {
			key := "spritesheet:" + anim.Current
			if c := go_.GetComponent(key); c != nil {
				if u, ok := c.(Updater); ok {
					u.Update(dt)
				}
			}
			continue
		}
		for _, c := range go_.components {
			if u, ok := c.(Updater); ok {
				u.Update(dt)
			}
		}
	}
}

// Draw runs Draw(screen, transform) on Drawer components of active GameObjects.
// If a GameObject has an Animator, only the spritesheet matching Animator.Current is drawn ("spritesheet:"+Current).
// Otherwise all Drawer components are drawn.
func (w *Manager) Draw(screen *ebiten.Image) {
	for _, go_ := range w.gameObjects {
		if !go_.Active {
			continue
		}
		// FIXME: this animator IF is leaking logic over here, should be refactored
		//  to be encapsuled in an Animator class or some kind of AnimatorManager
		transform := go_.Transform()
		anim := go_.Animator()
		if anim != nil && anim.Current != "" {
			key := "spritesheet:" + anim.Current
			if c := go_.GetComponent(key); c != nil {
				if d, ok := c.(Drawer); ok {
					d.Draw(screen, transform)
				}
			}
			continue
		}
		for _, c := range go_.components {
			if d, ok := c.(Drawer); ok {
				d.Draw(screen, transform)
			}
		}
	}
}
