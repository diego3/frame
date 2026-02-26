package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Manager maps action names to keys and exposes JustPressed/Pressed queries.
type Manager struct {
	bindings map[string][]ebiten.Key
}

// NewManager returns a manager with no bindings. Use Bind or SetBindings to configure.
func NewManager() *Manager {
	return &Manager{bindings: make(map[string][]ebiten.Key)}
}

// Bind assigns one or more keys to an action. Overwrites any existing bindings for the action.
func (m *Manager) Bind(action string, keys ...ebiten.Key) {
	m.bindings[action] = append([]ebiten.Key(nil), keys...)
}

// SetBindings replaces all bindings with the given map (action -> keys).
func (m *Manager) SetBindings(bindings map[string][]ebiten.Key) {
	m.bindings = make(map[string][]ebiten.Key)
	for action, keys := range bindings {
		m.bindings[action] = append([]ebiten.Key(nil), keys...)
	}
}

// ActionJustPressed returns true if any key bound to action was just pressed this frame.
func (m *Manager) ActionJustPressed(action string) bool {
	for _, key := range m.bindings[action] {
		if inpututil.IsKeyJustPressed(key) {
			return true
		}
	}
	return false
}

// ActionPressed returns true if any key bound to action is currently held.
func (m *Manager) ActionPressed(action string) bool {
	for _, key := range m.bindings[action] {
		if ebiten.IsKeyPressed(key) {
			return true
		}
	}
	return false
}

// defaultManager is the global instance used by package-level functions.
var defaultManager = NewManager()

func init() {
	// Default bindings for knight demo: toggle run, attack, attack2
	defaultManager.Bind("toggle_run", ebiten.KeySpace)
	defaultManager.Bind("dash", ebiten.KeyShiftLeft, ebiten.KeyShiftRight)
	defaultManager.Bind("attack", ebiten.KeyJ)
	defaultManager.Bind("attack2", ebiten.KeyK)
	// WASD movement
	defaultManager.Bind("move_left", ebiten.KeyA)
	defaultManager.Bind("move_right", ebiten.KeyD)
	defaultManager.Bind("move_up", ebiten.KeyW)
	defaultManager.Bind("move_down", ebiten.KeyS)
}

// ActionJustPressed reports whether any key for the given action was just pressed (uses default manager).
func ActionJustPressed(action string) bool {
	return defaultManager.ActionJustPressed(action)
}

// ActionPressed reports whether any key for the given action is currently held (uses default manager).
func ActionPressed(action string) bool {
	return defaultManager.ActionPressed(action)
}

// DefaultManager returns the global Manager so callers can rebind or inject a different one at startup.
func DefaultManager() *Manager {
	return defaultManager
}
