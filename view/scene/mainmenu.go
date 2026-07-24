package scene

import (
	"image/color"
	"path/filepath"

	"goengine/application/data"
	"goengine/event"
	"goengine/events"
	"goengine/object"
	"goengine/physics"
	"goengine/physics/box2d"
	"goengine/ports"
	"goengine/resource"
	"goengine/script"
	"goengine/view/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

	lua "github.com/yuin/gopher-lua"
)

// MainMenuState holds only simulation data for the main menu. Logic updates it in Update(dt) from intent events;
// View (Draw) only reads it to present. No rendering or input types.
type MainMenuState struct {
	World            *object.Manager
	DebugDrawPhysics bool
}

// MainMenu is the initial scene (title + click me button + data-driven GameObject world).
// Logic (Update) runs script components (shared VM per scene), then physics, then world update.
type MainMenu struct {
	titleImg         *ebiten.Image
	uiFace           font.Face
	world            *object.Manager
	vm               *script.VM
	loadedScripts    map[string]bool // path -> true once DoFile'd
	gameRoot         string          // base path for script DoFile (e.g. "games/demo1")
	physicsSystem    *PhysicsSystem
	debugDrawPhysics bool
}

// NewMainMenu returns a new main menu scene.
func NewMainMenu() *MainMenu {
	return &MainMenu{}
}

// Setup loads assets and builds the UI. Implements ports.Scene.
// If config has scene_path set, the world is built from that YAML; otherwise an empty world is used.
// ctx.Bus is used to emit intents (e.g. SceneChangeRequested) and to subscribe to events (e.g. DebugOverlayToggled).
func (m *MainMenu) Setup(ctx *ports.SceneContext) error {
	cfg, loader, root, bus := ctx.Config, ctx.Loader, ctx.UI, ctx.Bus

	event.Subscribe(bus, func(ev events.DebugOverlayToggled) {
		m.debugDrawPhysics = !m.debugDrawPhysics
	})
	event.Subscribe(bus, func(ev events.MoveRequested) {
		if m.world != nil {
			if k := m.world.Find("knight"); k != nil {
				if c := k.GetComponent("intent_buffer"); c != nil {
					ib := c.(*object.IntentBuffer)
					ib.PendingMoveX, ib.PendingMoveY = ev.DirX, ev.DirY
				}
			}
		}
	})
	event.Subscribe(bus, func(ev events.ScriptEmitted) {
		if m.vm != nil && m.vm.L != nil {
			_ = script.CallOnEvent(m.vm.L, ev.Name, ev.Payload)
		}
	})

	m.gameRoot = cfg.GameRoot
	if setter, ok := loader.(interface{ SetRoot(string) }); ok {
		setter.SetRoot(cfg.GameRoot)
	}
	m.vm = script.NewVM()
	m.loadedScripts = make(map[string]bool)
	playSound := func(path string) {
		_ = loader.LoadAudio(path)
		if p, err := loader.NewAudioPlayer(path); err == nil {
			p.Play()
		}
	}
	switchScene := func(sceneID string) { bus.Emit(events.SceneChangeRequested{SceneID: sceneID}) }
	quit := func() { bus.Emit(events.QuitRequested{}) }
	emit := func(name string, payload map[string]interface{}) {
		bus.Emit(events.ScriptEmitted{Name: name, Payload: payload})
	}
	m.vm.RegisterEngine("engine", script.EngineFuncs(playSound, switchScene, quit, emit))

	a := &cfg.Assets
	if err := loader.LoadFont(a.FontPath); err != nil {
		return err
	}
	face, err := loader.GetFace(a.FontPath, a.FontSize)
	if err != nil {
		return err
	}
	m.uiFace = face
	m.titleImg = resource.TextToImage(face, "Hello, Ebitengine!", color.White)

	_ = loader.LoadAudio(a.ClickSound)

	clickSound := a.ClickSound
	// TODO ui screen will be mounted from file as well
	root.AddElement(&ui.Button{
		X: 100, Y: 80, Width: 120, Height: 40,
		Label: "Click me",
		OnClick: func() {
			if p, err := loader.NewAudioPlayer(clickSound); err == nil {
				p.Play()
			}
		},
	})

	// Data-driven world: load scene from YAML if path set (path relative to game root)
	if a.ScenePath != "" {
		scenePath := a.ScenePath
		if m.gameRoot != "" {
			scenePath = filepath.Join(m.gameRoot, scenePath)
		}
		def, err := data.LoadScene(scenePath)
		if err != nil {
			return err
		}
		m.world, err = data.BuildWorld(def, loader)
		if err != nil {
			return err
		}
	} else {
		m.world = object.NewManager()
	}

	// Physics (anti-corruption: only box2d import here; game code uses physics.* only)
	if m.world != nil {
		gravity := physics.Vec2{X: cfg.Physics.GravityX, Y: cfg.Physics.GravityY}
		pixelScale := cfg.Physics.PixelScale
		physWorld := box2d.NewWorld(gravity, pixelScale)
		m.physicsSystem = NewPhysicsSystem(physWorld)
		m.physicsSystem.InitFromWorld(m.world)
		m.physicsSystem.LogBodies()
	}

	return nil
}

// Update implements ports.Scene. Runs script components (shared VM), then physics step, sync, world update.
func (m *MainMenu) Update(dt float64) {
	if m.world == nil || m.vm == nil {
		return
	}
	m.updateScripts(dt)
	if m.physicsSystem != nil {
		m.physicsSystem.Step(dt)
		m.physicsSystem.SyncToWorld(m.world)
	}
	m.world.Update(dt)
}

// updateScripts runs the shared Lua VM's update(dt) (or Script.UpdateFuncName) for every active
// GameObject with a script component, loading each script file on first use.
func (m *MainMenu) updateScripts(dt float64) {
	for _, go_ := range m.world.Objects() {
		if !go_.Active {
			continue
		}
		sc := go_.GetComponent("script")
		if sc == nil {
			continue
		}
		s, ok := sc.(*object.Script)
		if !ok || s.Path == "" {
			continue
		}
		if !m.loadedScripts[s.Path] {
			scriptPath := s.Path
			if m.gameRoot != "" {
				scriptPath = filepath.Join(m.gameRoot, scriptPath)
			}
			if err := m.vm.DoFile(scriptPath); err != nil {
				continue
			}
			m.loadedScripts[s.Path] = true
		}
		funcName := s.UpdateFuncName
		if funcName == "" {
			funcName = "update"
		}
		script.SetSelf(m.vm.L, go_)
		if _, err := m.vm.CallFunc(funcName, lua.LNumber(dt)); err != nil {
			continue
		}
	}
}

// Draw renders the scene from current state and view assets. Implements ports.Scene.
// View only reads state (MainMenuState) and does not mutate simulation data.
func (m *MainMenu) Draw(screen *ebiten.Image) {
	state := m.state()
	if m.titleImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(20, 20)
		screen.DrawImage(m.titleImg, op)
	}
	if state.World != nil {
		state.World.Draw(screen)
		if state.DebugDrawPhysics && m.physicsSystem != nil {
			m.physicsSystem.DrawDebug(screen, state.World)
		}
	}
}

// state returns the current simulation state for the view to read (Logic updates these fields in Update).
func (m *MainMenu) state() MainMenuState {
	return MainMenuState{
		World:            m.world,
		DebugDrawPhysics: m.debugDrawPhysics,
	}
}

// UIFace returns the font face for UI labels. Implements ports.Scene.
func (m *MainMenu) UIFace() font.Face {
	return m.uiFace
}
