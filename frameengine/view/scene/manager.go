package scene

import (
	"fmt"
	"sync"

	"goengine/frameengine/application/config"
	"goengine/frameengine/event"
	"goengine/frameengine/ports"
)

// SceneFactory creates a new scene instance. Used by Manager to instantiate scenes by id.
type SceneFactory func() (ports.Scene, error)

// Manager holds a registry of scenes by id and the current active scene.
// Register scenes with Register; switch with SwitchTo. No teardown of the previous scene.
type Manager struct {
	mu        sync.RWMutex
	factories map[string]SceneFactory
	current   ports.Scene
}

// NewManager returns an empty scene manager. Register scenes then call SwitchTo to set the first scene.
func NewManager() *Manager {
	return &Manager{
		factories: make(map[string]SceneFactory),
	}
}

// Register adds a scene factory for the given id. Overwrites any existing registration.
func (m *Manager) Register(id string, factory SceneFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.factories[id] = factory
}

// SwitchTo creates a new scene from the registered factory for id, calls Setup, and sets it as current.
// Returns an error if the id is unknown or scene creation/Setup fails.
func (m *Manager) SwitchTo(id string, cfg *config.Config, loader ports.AssetLoader, ui ports.UIRoot, bus *event.Bus) error {
	m.mu.Lock()
	factory, ok := m.factories[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("scene: unknown scene id %q", id)
	}
	sc, err := factory()
	if err != nil {
		return fmt.Errorf("scene: create %q: %w", id, err)
	}
	ctx := &ports.SceneContext{Config: cfg, Loader: loader, UI: ui, Bus: bus}
	if err := sc.Setup(ctx); err != nil {
		return fmt.Errorf("scene: setup %q: %w", id, err)
	}
	m.mu.Lock()
	m.current = sc
	m.mu.Unlock()
	return nil
}

// CurrentScene returns the active scene, or nil if none has been set (e.g. before first SwitchTo).
func (m *Manager) CurrentScene() ports.Scene {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}
