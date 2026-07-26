package scene

import (
	"image/color"
	"io/fs"
	"math"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

	"goengine/application/config"
	"goengine/application/data"
	"goengine/event"
	"goengine/events"
	"goengine/object"
	"goengine/physics"
	"goengine/physics/box2d"
	"goengine/ports"
	"goengine/resource"
	"goengine/script"
	"goengine/view/camera"
	"goengine/view/ui"
)

// MainMenuState holds only simulation data for the main menu. Logic updates it in Update(dt) from intent events;
// View (Draw) only reads it to present. No rendering or input types.
type MainMenuState struct {
	World            *object.Manager
	DebugDrawPhysics bool
}

// MainMenu is the initial scene (title + click me button + data-driven GameObject world).
// Logic (Update) runs script components (shared engine per scene), then physics, then world update.
type MainMenu struct {
	titleImg         *ebiten.Image
	uiFace           font.Face
	world            *object.Manager
	engine           script.Engine
	loadedScripts    map[string]bool // path -> true once loaded
	gameRoot         string          // base path for script loading on OS filesystem (e.g. "games/demo1")
	fsys             fs.FS           // when non-nil, scripts and scenes are loaded from this FS instead
	physicsSystem    *PhysicsSystem
	debugDrawPhysics bool
	cam              *camera.Camera // nil unless cfg.Camera.Follow is set
	camTarget        string         // name of the GameObject the camera follows
	worldBuffer      *ebiten.Image  // offscreen sized to the level; drawn to screen translated by -cam.X/Y
}

// NewMainMenu returns a new main menu scene.
func NewMainMenu() *MainMenu {
	return &MainMenu{}
}

// Setup loads assets and builds the UI. Implements ports.Scene.
// If config has scene_path set, the world is built from that YAML; otherwise an empty world is used.
// bus is used to emit intents (e.g. SceneChangeRequested) and to subscribe to events (e.g. DebugOverlayToggled).
// The script engine backend (Lua or Python) is selected from cfg.ScriptEngine.
func (m *MainMenu) Setup(cfg *config.Config, loader ports.AssetLoader, root ports.UIRoot, bus *event.Bus) error {
	event.Subscribe(bus, func(ev events.DebugOverlayToggled) {
		m.debugDrawPhysics = !m.debugDrawPhysics
	})
	event.Subscribe(bus, func(ev events.MoveRequested) {
		if controlled := m.findControlled(); controlled != nil {
			if c := controlled.GetComponent("intent_buffer"); c != nil {
				ib := c.(*object.IntentBuffer)
				ib.PendingMoveX, ib.PendingMoveY = ev.DirX, ev.DirY
			}
		}
	})
	event.Subscribe(bus, func(ev events.ScriptEmitted) {
		if ev.Name == "SpawnProjectile" {
			m.spawnProjectile(ev.Payload)
		}
		if m.engine != nil {
			_ = m.engine.CallOnEvent(ev.Name, ev.Payload)
		}
	})

	m.gameRoot = cfg.GameRoot
	m.fsys = cfg.FS
	if setter, ok := loader.(interface{ SetRoot(string) }); ok {
		setter.SetRoot(cfg.GameRoot)
	}

	// Create the script engine for this scene (Lua or Python based on config).
	m.engine = script.NewEngine(cfg.ScriptEngine)
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
	m.engine.RegisterEngineAPI(playSound, switchScene, quit, emit)

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

	// Data-driven world: load scene from YAML if path set (path relative to game root)
	if a.ScenePath != "" {
		var def *data.SceneDef
		var err error
		if m.fsys != nil {
			def, err = data.LoadSceneFS(m.fsys, a.ScenePath)
		} else {
			scenePath := a.ScenePath
			if m.gameRoot != "" {
				scenePath = filepath.Join(m.gameRoot, scenePath)
			}
			def, err = data.LoadScene(scenePath)
		}
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

	// Camera-follow: only set up if the scene opts in via cfg.Camera.Follow.
	if cfg.Camera.Follow != "" {
		levelW, levelH := cfg.Layout.LevelWidth, cfg.Layout.LevelHeight
		if levelW <= 0 {
			levelW = cfg.Layout.Width
		}
		if levelH <= 0 {
			levelH = cfg.Layout.Height
		}
		m.cam = camera.New(cfg.Layout.Width, cfg.Layout.Height, levelW, levelH)
		m.camTarget = cfg.Camera.Follow
		m.worldBuffer = ebiten.NewImage(levelW, levelH)
	}

	return nil
}

// findControlled returns the first active GameObject with an intent_buffer component — the single
// player-controlled entity a scene is expected to have (e.g. "knight" in demo1, "player" in
// metalslug_demo). Not name-specific, so scenes are free to name their controlled entity anything.
func (m *MainMenu) findControlled() *object.GameObject {
	if m.world == nil {
		return nil
	}
	for _, go_ := range m.world.Objects() {
		if go_.Active && go_.GetComponent("intent_buffer") != nil {
			return go_
		}
	}
	return nil
}

// spawnProjectile builds a projectile GameObject at the controlled entity's center, offset in the
// facing direction given by payload's "dir_x"/"dir_y" (defaults to facing right), and adds it to
// the world. Movement and off-screen deactivation aren't implemented yet — see
// docs/game_concept_metal_slug_demo.md build order step 4 — so the projectile is visible but
// stationary until that lands.
func (m *MainMenu) spawnProjectile(payload map[string]interface{}) {
	if m.world == nil {
		return
	}
	origin := m.findControlled()
	if origin == nil {
		return
	}
	t := origin.Transform()
	if t == nil {
		return
	}
	cx, cy := t.X, t.Y
	if pb := origin.PhysicsBody(); pb != nil {
		cx += pb.Width / 2
		cy += pb.Height / 2
	}

	dirX, dirY := normalizeDir(payloadFloat(payload, "dir_x", 1), payloadFloat(payload, "dir_y", 0))

	const (
		projectileSpeed  = 360.0 // world units/sec, consumed once movement lands
		projectileDamage = 1.0
		projectileSize   = 8.0
		spawnOffset      = 30.0 // clear of the controlled entity's own body
	)

	proj := object.NewGameObject("projectile")
	proj.AddComponent(&object.Transform{
		X: cx + dirX*spawnOffset - projectileSize/2,
		Y: cy + dirY*spawnOffset - projectileSize/2,
	})
	proj.AddComponent(&object.Block{Width: projectileSize, Height: projectileSize})
	proj.AddComponent(&object.Projectile{VelX: dirX * projectileSpeed, VelY: dirY * projectileSpeed, Damage: projectileDamage})
	m.world.Add(proj)
}

// payloadFloat reads key from payload as a float64, accepting whatever numeric type the script
// engine produced (Lua and Python payload numbers may arrive as float64, int, or int64), or
// fallback if key is absent or not a number.
func payloadFloat(payload map[string]interface{}, key string, fallback float64) float64 {
	switch v := payload[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return fallback
}

// normalizeDir returns (x, y) scaled to unit length, or (1, 0) if both are zero.
func normalizeDir(x, y float64) (float64, float64) {
	length := math.Sqrt(x*x + y*y)
	if length == 0 {
		return 1, 0
	}
	return x / length, y / length
}

// cameraTargetCenter returns the world-space center of the camera's follow target, and whether it
// was found. Center is the transform position plus half the physics body size when present,
// falling back to the raw transform position (top-left) otherwise.
func (m *MainMenu) cameraTargetCenter() (x, y float64, ok bool) {
	if m.world == nil || m.camTarget == "" {
		return 0, 0, false
	}
	obj := m.world.Find(m.camTarget)
	if obj == nil {
		return 0, 0, false
	}
	t := obj.Transform()
	if t == nil {
		return 0, 0, false
	}
	x, y = t.X, t.Y
	if pb := obj.PhysicsBody(); pb != nil {
		x += pb.Width / 2
		y += pb.Height / 2
	}
	return x, y, true
}

// Update implements ports.Scene. Runs script components (shared engine), then physics step, sync, world update.
func (m *MainMenu) Update(dt float64) {
	if m.world == nil || m.engine == nil {
		return
	}
	m.updateScripts(dt)
	if m.physicsSystem != nil {
		m.physicsSystem.Step(dt)
		m.physicsSystem.SyncToWorld(m.world)
	}
	m.world.Update(dt)
	if m.cam != nil {
		if cx, cy, ok := m.cameraTargetCenter(); ok {
			m.cam.Follow(cx, cy)
		}
	}
}

// updateScripts runs the script engine's update(dt) (or Script.UpdateFuncName) for every active
// GameObject with a script component, loading each script file (or FS-embedded source) on first use.
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
		scriptPath := s.Path
		if m.gameRoot != "" {
			scriptPath = filepath.Join(m.gameRoot, scriptPath)
		}
		if !m.loadedScripts[s.Path] {
			var loadErr error
			if m.fsys != nil {
				src, readErr := fs.ReadFile(m.fsys, s.Path)
				if readErr != nil {
					continue
				}
				loadErr = m.engine.DoString(scriptPath, string(src))
			} else {
				loadErr = m.engine.DoFile(scriptPath)
			}
			if loadErr != nil {
				continue
			}
			m.loadedScripts[s.Path] = true
		}
		funcName := s.UpdateFuncName
		if funcName == "" {
			funcName = "update"
		}
		if err := m.engine.CallScriptUpdate(scriptPath, funcName, go_, dt); err != nil {
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
		if m.cam != nil {
			// World objects draw at absolute world coordinates, which may exceed the visible
			// viewport once a level is wider/taller than the screen. Draw them onto an
			// offscreen buffer sized to the level, then blit that buffer onto the real screen
			// translated by -cam.X/-cam.Y so only the camera's viewport is visible.
			m.worldBuffer.Clear()
			state.World.Draw(m.worldBuffer)
			if state.DebugDrawPhysics && m.physicsSystem != nil {
				m.physicsSystem.DrawDebug(m.worldBuffer, state.World)
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-m.cam.X, -m.cam.Y)
			screen.DrawImage(m.worldBuffer, op)
		} else {
			state.World.Draw(screen)
			if state.DebugDrawPhysics && m.physicsSystem != nil {
				m.physicsSystem.DrawDebug(screen, state.World)
			}
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
