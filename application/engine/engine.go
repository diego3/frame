package engine

import (
	"os"
	"os/signal"
	"syscall"

	"goengine/application/config"
	"goengine/application/game"
	"goengine/event"
	"goengine/ports"
	"goengine/resource"
	"goengine/view/scene"
	"goengine/view/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// Engine composes config, resources, UI, scene, and game loop. Create with New, then call Run.
type Engine struct {
	cfg  *config.Config
	game *game.Game
}

// New builds the dependency graph and returns an Engine. cfg must not be nil.
func New(cfg *config.Config) *Engine {
	bus := event.NewBus()
	var loader ports.AssetLoader = resource.NewManager()
	u := ui.NewContainer()
	sm := scene.NewManager()
	// FIXME: this is a problem, add a new scene here is not ideal, it should be data driven
	sm.Register("main_menu", func() (ports.Scene, error) { return scene.NewMainMenu(), nil })
	g := game.New(cfg, loader, u, sm, "main_menu", bus)
	return &Engine{cfg: cfg, game: g}
}

// Shutdown releases resources. Safe to call multiple times. Call from main (e.g. defer e.Shutdown()).
func (e *Engine) Shutdown() {
	if e.game != nil {
		e.game.Shutdown()
	}
}

// Run initializes the game, applies window settings, and runs the ebiten loop.
// On SIGINT (Ctrl+C) or SIGTERM, the game loop exits cleanly and Run returns game.ErrShutdownRequested.
// The caller should defer e.Shutdown() before Run so cleanup runs on any exit path.
func (e *Engine) Run() error {
	if err := e.game.Init(); err != nil {
		return err
	}

	shutdownCh := make(chan struct{})
	e.game.SetShutdownChannel(shutdownCh)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		signal.Stop(sigCh)
		close(shutdownCh)
	}()

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
