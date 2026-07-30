package engine

import (
	"goengine/frameengine/application/config"
	"goengine/frameengine/application/game"
	"goengine/frameengine/event"
	"goengine/frameengine/ports"
	"goengine/frameengine/resource"
	"goengine/frameengine/view/scene"
	"goengine/frameengine/view/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// Engine composes config, resources, UI, scene, and game loop. Create with New, then call Run.
type Engine struct {
	cfg  *config.Config
	game *game.Game
}

// New builds the dependency graph and returns an Engine. cfg must not be nil.
// sceneFactories maps scene type name (as used in cfg's "scenes" map) to a constructor; the
// engine itself has no built-in scene types — the caller (typically main.go) supplies whichever
// scene types its game actually needs. This keeps scene registration data-driven and out of the
// engine's own global state (previously a package-level scene.Factories map).
func New(cfg *config.Config, sceneFactories map[string]scene.SceneFactory) *Engine {
	bus := event.NewBus()
	mgr := resource.NewManager()
	if cfg.FS != nil {
		mgr.SetFS(cfg.FS)
	}
	var loader ports.AssetLoader = mgr
	u := ui.NewContainer()
	sm := scene.NewManager()
	// Scenes are declared in config (id -> type); types are looked up in sceneFactories.
	// Unknown types are skipped here and surface as a clear "unknown scene id" error from
	// scene.Manager.SwitchTo when Init() tries to load them.
	for id, sceneType := range cfg.Scenes {
		if factory, ok := sceneFactories[sceneType]; ok {
			sm.Register(id, factory)
		}
	}
	g := game.New(cfg, loader, u, sm, cfg.InitialScene, bus)
	return &Engine{cfg: cfg, game: g}
}

// Shutdown releases resources. Safe to call multiple times. Call from main (e.g. defer e.Shutdown()).
func (e *Engine) Shutdown() {
	if e.game != nil {
		e.game.Shutdown()
	}
}

// Run initializes the game, applies window settings, and runs the ebiten loop.
// On SIGINT (Ctrl+C) or SIGTERM (non-WASM), the game loop exits cleanly and Run returns game.ErrShutdownRequested.
// The caller should defer e.Shutdown() before Run so cleanup runs on any exit path.
func (e *Engine) Run() error {
	if err := e.game.Init(); err != nil {
		return err
	}

	shutdownCh := make(chan struct{})
	e.game.SetShutdownChannel(shutdownCh)
	watchSignals(shutdownCh)

	w := e.cfg.Window
	ebiten.SetWindowSize(w.Width, w.Height)
	ebiten.SetWindowTitle(w.Title)
	if w.Resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	} else {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	}
	if w.Fullscreen {
		ebiten.SetFullscreen(true)
	}

	return ebiten.RunGame(e.game)
}
