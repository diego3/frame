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

// DefaultBindings returns the default action->keys map. Used to initialize a Manager
// before applying game config overrides (see ManagerFromConfig).
func DefaultBindings() map[string][]ebiten.Key {
	return map[string][]ebiten.Key{
		"debug_overlay": {ebiten.KeyF3},
		"toggle_run":    {ebiten.KeySpace},
		"dash":          {ebiten.KeyShiftLeft, ebiten.KeyShiftRight},
		"attack":        {ebiten.KeyJ},
		"attack2":       {ebiten.KeyK},
		"move_left":     {ebiten.KeyA},
		"move_right":    {ebiten.KeyD},
		"move_up":       {ebiten.KeyW},
		"move_down":     {ebiten.KeyS},
	}
}

func init() {
	defaultManager.SetBindings(DefaultBindings())
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
