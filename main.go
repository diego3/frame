//go:build !js

package main

import (
	"errors"
	"log"
	"os"

	"goengine/application/config"
	"goengine/application/engine"
	"goengine/application/game"
)

func main() {
	configPath := "games/demo1/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("config: ", err)
	}

	e := engine.New(cfg)
	defer e.Shutdown()

	if err := e.Run(); err != nil {
		if errors.Is(err, game.ErrShutdownRequested) {
			os.Exit(0)
		}
		log.Fatal(err)
	}
}
