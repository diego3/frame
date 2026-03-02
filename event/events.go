package event

// TODO, this events shoud be moved to its own package

// SceneChangeRequested is emitted when a scene or UI requests a switch to another scene.
// Subscribers (e.g. application) perform the actual switch.
type SceneChangeRequested struct {
	SceneID string
}

// QuitRequested is emitted when the application should exit (e.g. user chose Quit).
// Subscribers (e.g. application) perform shutdown.
type QuitRequested struct{}

// SceneChanged is a state event emitted after the application has switched to a new scene.
// Subscribers (e.g. views) can react to present the new screen.
type SceneChanged struct {
	SceneID string
}

// DebugOverlayToggled is an intent event emitted when the user requests toggling the debug overlay (e.g. F3).
// Scenes (or a debug system) subscribe and toggle their debug draw state.
type DebugOverlayToggled struct{}

// MoveRequested is an intent event emitted each frame while the user holds a movement direction.
// Game Logic (or scene logic) subscribes and applies movement to the controlled entity.
type MoveRequested struct {
	DirX, DirY float64 // normalized direction, e.g. -1/0/1
}

// DashRequested is an intent event emitted when the user requests a dash (e.g. Shift just pressed).
type DashRequested struct{}

// AttackRequested is an intent event emitted when the user requests primary attack (e.g. J just pressed).
type AttackRequested struct{}

// Attack2Requested is an intent event emitted when the user requests secondary attack (e.g. K just pressed).
type Attack2Requested struct{}

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
