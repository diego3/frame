// Package demo1 exposes the embedded assets for the demo1 game.
// Used by the WASM build so all game data is bundled into the binary.
package demo1

import "embed"

// FS contains all game data for demo1: config, scenes, scripts, and assets.
// Paths inside the FS are relative to the games/demo1 directory
// (e.g. "assets/ShadeBlue-2OozX.ttf", "scenes/main_menu.yaml").
//
//go:embed config.yaml scenes scripts assets
var FS embed.FS
