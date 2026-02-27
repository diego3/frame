package event

// SceneChangeRequested is emitted when a scene or UI requests a switch to another scene.
// Subscribers (e.g. application) perform the actual switch.
type SceneChangeRequested struct {
	SceneID string
}

// QuitRequested is emitted when the application should exit (e.g. user chose Quit).
// Subscribers (e.g. application) perform shutdown.
type QuitRequested struct{}

// GameObjectCreated is emitted when a new GameObject is created and added to a world.
// ID is a stable identifier; Name is the initial name (may change later).
type GameObjectCreated struct {
	ID   uint64
	Name string
}

// GameObjectDestroyed is emitted when a GameObject is removed from a world and will no longer update/draw.
type GameObjectDestroyed struct {
	ID   uint64
	Name string
}

// GameObjectActivated is emitted when a GameObject becomes active (e.g. Active flips from false to true).
type GameObjectActivated struct {
	ID uint64
}

// GameObjectDeactivated is emitted when a GameObject becomes inactive (e.g. Active flips from true to false).
type GameObjectDeactivated struct {
	ID uint64
}

// ComponentAdded is emitted when a component is attached to a GameObject.
// ComponentType matches the key used in the GameObject's component map (e.g. "transform", "physics_body").
type ComponentAdded struct {
	GameObjectID  uint64
	ComponentType string
}

// ComponentRemoved is emitted when a component is detached from a GameObject.
type ComponentRemoved struct {
	GameObjectID  uint64
	ComponentType string
}
