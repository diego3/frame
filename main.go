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
	cfg, err := config.Load("games/demo1/config.yaml")
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
