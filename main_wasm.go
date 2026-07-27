//go:build js && wasm

package main

import (
	"errors"
	"log"

	demo1 "goengine/games/demo1"

	"goengine/application/config"
	"goengine/application/engine"
	"goengine/application/game"
	"goengine/ports"
	"goengine/view/scene"
)

// sceneFactories are the scene types this WASM build (demo1 only) needs. See main.go's comment
// for why registration happens here rather than in the engine itself.
var sceneFactories = map[string]scene.SceneFactory{
	"world_scene": func() (ports.Scene, error) { return scene.NewWorldScene(), nil },
}

func main() {
	cfg, err := config.LoadFromFS(demo1.FS, "config.yaml")
	if err != nil {
		log.Fatal("config: ", err)
	}

	e := engine.New(cfg, sceneFactories)
	defer e.Shutdown()

	if err := e.Run(); err != nil {
		if errors.Is(err, game.ErrShutdownRequested) {
			return
		}
		log.Fatal(err)
	}
}
