//go:build !js

package main

import (
	"errors"
	"log"
	"os"

	"goengine/frameengine/application/config"
	"goengine/frameengine/application/engine"
	"goengine/frameengine/application/game"
	"goengine/frameengine/ports"
	"goengine/frameengine/view/scene"
	metalslug "goengine/games/metalslug_demo"
)

// sceneFactories are this application's scene types, keyed by the name used in a game's
// config.yaml "scenes" map. Registering both here (rather than in the engine itself) is what
// keeps scene registration data-driven per-application instead of hardcoded engine-side.
var sceneFactories = map[string]scene.SceneFactory{
	"world_scene":     func() (ports.Scene, error) { return scene.NewWorldScene(), nil },
	"metalslug_scene": func() (ports.Scene, error) { return metalslug.NewScene(), nil },
}

func main() {
	configPath := "games/metalslug_demo/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("config: ", err)
	}

	e := engine.New(cfg, sceneFactories)
	defer e.Shutdown()

	if err := e.Run(); err != nil {
		if errors.Is(err, game.ErrShutdownRequested) {
			os.Exit(0)
		}
		log.Fatal(err)
	}
}
