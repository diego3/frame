package scene

import (
	"image/color"

	"goengine/application/config"
	"goengine/application/data"
	"goengine/physics"
	"goengine/physics/box2d"
	"goengine/ports"
	"goengine/resource"
	"goengine/object"
	"goengine/view/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
)

// MainMenu is the initial scene (title + click me button + data-driven GameObject world).
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
// switcher can be used later to call SwitchTo("other_scene_id") (e.g. from a Start button).
func (m *MainMenu) Setup(cfg *config.Config, loader ports.AssetLoader, root ports.UIRoot, switcher ports.SceneSwitcher) error {
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
func (m *MainMenu) Update(dt float64) {
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		m.debugDrawPhysics = !m.debugDrawPhysics
	}
	if m.world != nil && m.knightController != nil {
		m.knightController.Update(m.world, dt)
		if m.physicsSystem != nil {
			m.physicsSystem.Step(dt)
			m.physicsSystem.SyncToWorld(m.world)
		}
		m.world.Update(dt)
	}
}

// Draw renders the scene. Implements ports.Scene.
func (m *MainMenu) Draw(screen *ebiten.Image) {
	if m.titleImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(20, 20)
		screen.DrawImage(m.titleImg, op)
	}
	if m.world != nil {
		m.world.Draw(screen)
		if m.debugDrawPhysics && m.physicsSystem != nil {
			m.physicsSystem.DrawDebug(screen, m.world)
		}
	}
}

// UIFace returns the font face for UI labels. Implements ports.Scene.
func (m *MainMenu) UIFace() font.Face {
	return m.uiFace
}
