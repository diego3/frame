package event

// SceneChangeRequested is emitted when a scene or UI requests a switch to another scene.
// Subscribers (e.g. application) perform the actual switch.
type SceneChangeRequested struct {
	SceneID string
}

// QuitRequested is emitted when the application should exit (e.g. user chose Quit).
// Subscribers (e.g. application) perform shutdown.
type QuitRequested struct{}
