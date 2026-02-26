package main

import (
	"errors"
	"log"
	"os"

	"goengine/config"
	"goengine/engine"
	"goengine/game"
)

func main() {
	cfg, err := config.Load("config.yaml")
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
