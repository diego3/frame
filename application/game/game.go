package game

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"goengine/application/config"
	"goengine/ports"
	"goengine/view/scene"
)

// ErrShutdownRequested is returned when the process received SIGINT/SIGTERM and the game loop exited cleanly.
var ErrShutdownRequested = errors.New("shutdown requested")

// Game holds the game state and implements ebiten.Game. It orchestrates config, loader, UI, and scene manager.
// Implements ports.SceneSwitcher so scenes can request a switch via SwitchTo(id).
type Game struct {
	cfg            *config.Config
	loader         ports.AssetLoader
	ui             ports.UIRoot
	manager        *scene.Manager
	initialSceneID string
	shutdownCh     <-chan struct{}
}

// New returns a new Game with the given dependencies. initialSceneID is the scene to load in Init() (e.g. "main_menu").
// Call Init() before RunGame.
func New(cfg *config.Config, loader ports.AssetLoader, ui ports.UIRoot, manager *scene.Manager, initialSceneID string) *Game {
	return &Game{
		cfg:            cfg,
		loader:         loader,
		ui:             ui,
		manager:        manager,
		initialSceneID: initialSceneID,
	}
}

// SwitchTo implements ports.SceneSwitcher. Switches to the registered scene by id.
func (g *Game) SwitchTo(sceneID string) error {
	return g.manager.SwitchTo(sceneID, g.cfg, g.loader, g.ui, g)
}

// SetShutdownChannel sets the channel that is closed when a graceful shutdown is requested (e.g. SIGINT/SIGTERM).
// The engine calls this before Run. If nil, no signal-based shutdown is performed.
func (g *Game) SetShutdownChannel(ch <-chan struct{}) {
	g.shutdownCh = ch
}

// Init runs once before the game loop. Loads the initial scene via the manager.
func (g *Game) Init() error {
	return g.manager.SwitchTo(g.initialSceneID, g.cfg, g.loader, g.ui, g)
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
