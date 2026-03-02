package scene

import (
	"image/color"

	"goengine/application/config"
	"goengine/application/data"
	"goengine/event"
	"goengine/object"
	"goengine/physics"
	"goengine/physics/box2d"
	"goengine/ports"
	"goengine/resource"
	"goengine/view/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
)

// MainMenuState holds only simulation data for the main menu. Logic updates it in Update(dt) from intent events;
// View (Draw) only reads it to present. No rendering or input types.
type MainMenuState struct {
	World            *object.World
	DebugDrawPhysics bool
}

// MainMenu is the initial scene (title + click me button + data-driven GameObject world).
// Logic (Update) updates state from intents; View (Draw) reads state and view assets only.
type MainMenu struct {
	titleImg         *ebiten.Image
	uiFace           font.Face
	world            *object.World
	knightController *KnightController
	physicsSystem    *PhysicsSystem
	debugDrawPhysics bool // F3 toggles collision box overlay
}

// NewMainMenu returns a new main menu scene.
func NewMainMenu() *MainMenu {
	return &MainMenu{}
}

// Setup loads assets and builds the UI. Implements ports.Scene.
// If config has scene_path set, the world is built from that YAML; otherwise an empty world is used.
// bus is used to emit intents (e.g. SceneChangeRequested) and to subscribe to events (e.g. DebugOverlayToggled).
func (m *MainMenu) Setup(cfg *config.Config, loader ports.AssetLoader, root ports.UIRoot, bus *event.Bus) error {
	event.Subscribe(bus, func(ev event.DebugOverlayToggled) {
		m.debugDrawPhysics = !m.debugDrawPhysics
	})
	event.Subscribe(bus, func(ev event.MoveRequested) {
		if m.knightController != nil {
			m.knightController.PendingMoveX, m.knightController.PendingMoveY = ev.DirX, ev.DirY
		}
	})
	event.Subscribe(bus, func(ev event.DashRequested) {
		if m.knightController != nil {
			m.knightController.PendingDash = true
		}
	})
	event.Subscribe(bus, func(ev event.AttackRequested) {
		if m.knightController != nil {
			m.knightController.PendingAttack = true
		}
	})
	event.Subscribe(bus, func(ev event.Attack2Requested) {
		if m.knightController != nil {
			m.knightController.PendingAttack2 = true
		}
	})
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
	root.AddButton(&ui.Button{
		X: 100, Y: 80, Width: 120, Height: 40,
		Label: "Click me",
		OnClick: func() {
			if p, err := loader.NewAudioPlayer(clickSound); err == nil {
				p.Play()
			}
		},
	})

	// Data-driven world: load scene from YAML if path set
	if a.ScenePath != "" {
		def, err := data.LoadScene(a.ScenePath)
		if err != nil {
			return err
		}
		m.world, err = data.BuildWorld(def, loader)
		if err != nil {
			return err
		}
	} else {
		m.world = object.NewWorld()
	}

	// Physics (anti-corruption: only box2d import here; game code uses physics.* only)
	// Bodies are created from scene data: objects with physics_body component get a body in InitFromWorld.
	if m.world != nil {
		m.knightController = &KnightController{}
		gravity := physics.Vec2{X: cfg.Physics.GravityX, Y: cfg.Physics.GravityY}
		pixelScale := cfg.Physics.PixelScale
		physWorld := box2d.NewWorld(gravity, pixelScale)
		m.physicsSystem = NewPhysicsSystem(physWorld)
		m.physicsSystem.InitFromWorld(m.world)
		m.physicsSystem.LogBodies()
	}

	return nil
}

// Update implements ports.Scene. Runs knight input, physics step, sync, then world update.
// Debug overlay toggle is driven by DebugOverlayToggled events (F3 is handled in Application layer).
func (m *MainMenu) Update(dt float64) {
	if m.world != nil && m.knightController != nil {
		m.knightController.Update(m.world, dt)
		if m.physicsSystem != nil {
			m.physicsSystem.Step(dt)
			m.physicsSystem.SyncToWorld(m.world)
		}
		m.world.Update(dt)
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
		World:           m.world,
		DebugDrawPhysics: m.debugDrawPhysics,
	}
}

// UIFace returns the font face for UI labels. Implements ports.Scene.
func (m *MainMenu) UIFace() font.Face {
	return m.uiFace
}
