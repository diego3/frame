package scene

import "goengine/ports"

// Factories maps a scene type name (as used in config.yaml's "scenes" map) to its constructor.
// Built-in scene types register themselves here so callers (e.g. engine.New) can build a
// scene.Manager purely from config, without hardcoding scene ids/types in Go.
var Factories = map[string]SceneFactory{
	"main_menu": func() (ports.Scene, error) { return NewMainMenu(), nil },
}
