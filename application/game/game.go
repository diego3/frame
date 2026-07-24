package game

import (
	"errors"
	"sync"

	"goengine/application/config"
	"goengine/event"
	"goengine/events"
	"goengine/ports"
	"goengine/script"
	"goengine/view/input"
	"goengine/view/scene"

	"github.com/hajimehoshi/ebiten/v2"
)

// ErrShutdownRequested is returned when the process received SIGINT/SIGTERM or QuitRequested and the game loop exited cleanly.
var ErrShutdownRequested = errors.New("shutdown requested")

// Game holds the game state and implements ebiten.Game. It orchestrates config, loader, UI, and scene manager.
// Scene change and quit are driven by events; Game subscribes to SceneChangeRequested and QuitRequested.
// Input is translated to intents by input.Adapter before scene Update.
// A script engine (Lua or Python, selected via config) is available via ScriptEngine();
// engine.play_sound, engine.switch_scene, engine.quit, and engine.emit are registered.
type Game struct {
	cfg            *config.Config
	loader         ports.AssetLoader
	ui             ports.UIRoot
	manager        *scene.Manager
	initialSceneID string
	bus            *event.Bus
	inputAdapter   *input.Adapter
	scriptEngine   script.Engine
	shutdownCh     <-chan struct{}
	quitCh         chan struct{}
	quitOnce       sync.Once
}

// New returns a new Game with the given dependencies. initialSceneID is the scene to load in Init() (e.g. "main_menu").
// bus is the event bus: Game subscribes to SceneChangeRequested and QuitRequested and passes bus to scenes.
// Input adapter uses bindings from cfg.Input (overriding defaults) and cfg.ScriptEvents for script event names.
func New(cfg *config.Config, loader ports.AssetLoader, ui ports.UIRoot, manager *scene.Manager, initialSceneID string, bus *event.Bus) *Game {
	mgr := input.NewManager()
	mgr.SetBindings(input.DefaultBindings())
	if cfg != nil && len(cfg.Input) > 0 {
		if parsed, err := input.ParseBindings(cfg.Input); err == nil {
			for action, keys := range parsed {
				mgr.Bind(action, keys...)
			}
		}
	}
	scriptEvents := cfg.ScriptEvents
	if scriptEvents == nil {
		scriptEvents = make(map[string]string)
	}
	g := &Game{
		cfg:            cfg,
		loader:         loader,
		ui:             ui,
		manager:        manager,
		initialSceneID: initialSceneID,
		bus:            bus,
		inputAdapter:   input.NewAdapter(mgr, event.NewIntentBus(bus), scriptEvents),
		quitCh:         make(chan struct{}),
	}
	event.Subscribe(bus, func(ev events.SceneChangeRequested) {
		if err := g.manager.SwitchTo(ev.SceneID, g.cfg, g.loader, g.ui, g.bus); err != nil {
			return
		}
		event.Emit(g.bus, events.SceneChanged{SceneID: ev.SceneID})
	})
	event.Subscribe(bus, func(ev events.QuitRequested) {
		g.quitOnce.Do(func() { close(g.quitCh) })
	})
	return g
}

// SetShutdownChannel sets the channel that is closed when a graceful shutdown is requested (e.g. SIGINT/SIGTERM).
// The engine calls this before Run. If nil, no signal-based shutdown is performed.
func (g *Game) SetShutdownChannel(ch <-chan struct{}) {
	g.shutdownCh = ch
}

// Init runs once before the game loop. Loads the initial scene via the manager and initializes the script engine.
// The backend (Lua or Python) is determined by cfg.ScriptEngine.
func (g *Game) Init() error {
	g.scriptEngine = script.NewEngine(g.cfg.ScriptEngine)
	g.scriptEngine.RegisterEngineAPI(
		g.playSound,
		g.switchScene,
		g.quit,
		g.emitScript,
	)
	if err := g.manager.SwitchTo(g.initialSceneID, g.cfg, g.loader, g.ui, g.bus); err != nil {
		return err
	}
	return nil
}

// playSound loads and plays the audio at path. Used by Lua engine.play_sound(path).
func (g *Game) playSound(path string) {
	_ = g.loader.LoadAudio(path)
	if p, err := g.loader.NewAudioPlayer(path); err == nil {
		p.Play()
	}
}

// switchScene emits SceneChangeRequested so the application switches scene. Used by Lua engine.switch_scene(id).
func (g *Game) switchScene(sceneID string) {
	g.bus.Emit(events.SceneChangeRequested{SceneID: sceneID})
}

// quit emits QuitRequested so the application exits. Used by Lua engine.quit().
func (g *Game) quit() {
	g.bus.Emit(events.QuitRequested{})
}

// emitScript emits ScriptEmitted so scripts can fire custom events via engine.emit(name, payload).
func (g *Game) emitScript(name string, payload map[string]interface{}) {
	g.bus.Emit(events.ScriptEmitted{Name: name, Payload: payload})
}

// Shutdown runs when the game exits. Releases loaded resources and closes the script engine.
func (g *Game) Shutdown() {
	if g.scriptEngine != nil {
		g.scriptEngine.Close()
		g.scriptEngine = nil
	}
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

// ScriptEngine returns the active script engine (Lua or Python).
// The engine has engine.play_sound, engine.switch_scene, engine.quit, and engine.emit registered.
// Returns nil after Shutdown.
func (g *Game) ScriptEngine() script.Engine {
	return g.scriptEngine
}
