package game

import (
	"errors"
	"sync"

	"goengine/application/config"
	"goengine/event"
	"goengine/ports"
	"goengine/view/input"
	"goengine/view/scene"

	"github.com/hajimehoshi/ebiten/v2"
)

// ErrShutdownRequested is returned when the process received SIGINT/SIGTERM or QuitRequested and the game loop exited cleanly.
var ErrShutdownRequested = errors.New("shutdown requested")

// Game holds the game state and implements ebiten.Game. It orchestrates config, loader, UI, and scene manager.
// Scene change and quit are driven by events; Game subscribes to SceneChangeRequested and QuitRequested.
// Input is translated to intents by input.Adapter before scene Update.
type Game struct {
	cfg            *config.Config
	loader         ports.AssetLoader
	ui             ports.UIRoot
	manager        *scene.Manager
	initialSceneID string
	bus            *event.Bus
	inputAdapter   *input.Adapter
	shutdownCh     <-chan struct{}
	quitCh         chan struct{}
	quitOnce       sync.Once
}

// New returns a new Game with the given dependencies. initialSceneID is the scene to load in Init() (e.g. "main_menu").
// bus is the event bus: Game subscribes to SceneChangeRequested and QuitRequested and passes bus to scenes.
// Input adapter polls keys and emits intents (MoveRequested, DebugOverlayToggled, etc.) so Logic does not read input directly.
func New(cfg *config.Config, loader ports.AssetLoader, ui ports.UIRoot, manager *scene.Manager, initialSceneID string, bus *event.Bus) *Game {
	g := &Game{
		cfg:            cfg,
		loader:         loader,
		ui:             ui,
		manager:        manager,
		initialSceneID: initialSceneID,
		bus:            bus,
		inputAdapter:   input.NewAdapter(input.DefaultManager(), event.NewIntentBus(bus)),
		quitCh:         make(chan struct{}),
	}
	event.Subscribe(bus, func(ev event.SceneChangeRequested) {
		if err := g.manager.SwitchTo(ev.SceneID, g.cfg, g.loader, g.ui, g.bus); err != nil {
			return
		}
		event.Emit(g.bus, event.SceneChanged{SceneID: ev.SceneID})
	})
	event.Subscribe(bus, func(ev event.QuitRequested) {
		g.quitOnce.Do(func() { close(g.quitCh) })
	})
	return g
}

// SetShutdownChannel sets the channel that is closed when a graceful shutdown is requested (e.g. SIGINT/SIGTERM).
// The engine calls this before Run. If nil, no signal-based shutdown is performed.
func (g *Game) SetShutdownChannel(ch <-chan struct{}) {
	g.shutdownCh = ch
}

// Init runs once before the game loop. Loads the initial scene via the manager.
func (g *Game) Init() error {
	return g.manager.SwitchTo(g.initialSceneID, g.cfg, g.loader, g.ui, g.bus)
}

// Shutdown runs when the game exits. Releases loaded resources.
func (g *Game) Shutdown() {
	g.loader.Clear()
}

// Update implements ebiten.Game.
func (g *Game) Update() error {
	if g.shutdownCh != nil {
		select {
		case <-g.shutdownCh:
			return ErrShutdownRequested
		default:
		}
	}

	select {
	case <-g.quitCh:
		return ErrShutdownRequested
	default:
	}

	g.inputAdapter.Poll()

	const defaultDt = 1.0 / 60.0
	if sc := g.manager.CurrentScene(); sc != nil {
		sc.Update(defaultDt)
	}

	g.ui.Update(g.cfg.Layout.Width, g.cfg.Layout.Height)
	return nil
}

// Draw implements ebiten.Game.
func (g *Game) Draw(screen *ebiten.Image) {
	if sc := g.manager.CurrentScene(); sc != nil {
		sc.Draw(screen)
		g.ui.Draw(screen, sc.UIFace())
	}
}

// Layout implements ebiten.Game.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.cfg.Layout.Width, g.cfg.Layout.Height
}
