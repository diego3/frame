//go:build js && wasm

package main

import (
	"errors"
	"log"

	demo1 "goengine/games/demo1"

	"goengine/application/config"
	"goengine/application/engine"
	"goengine/application/game"
)

func main() {
	cfg, err := config.LoadFromFS(demo1.FS, "config.yaml")
	if err != nil {
		log.Fatal("config: ", err)
	}

	e := engine.New(cfg)
	defer e.Shutdown()

	if err := e.Run(); err != nil {
		if errors.Is(err, game.ErrShutdownRequested) {
			return
		}
		log.Fatal(err)
	}
}
