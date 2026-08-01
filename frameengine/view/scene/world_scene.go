package scene

import (
	"image/color"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

	"goengine/frameengine/application/data"
	"goengine/frameengine/event"
	"goengine/frameengine/events"
	"goengine/frameengine/object"
	"goengine/frameengine/physics"
	"goengine/frameengine/physics/box2d"
	"goengine/frameengine/ports"
	"goengine/frameengine/process"
	"goengine/frameengine/resource"
	"goengine/frameengine/script"
	"goengine/frameengine/view/camera"
	"goengine/frameengine/view/ui"
)

// WorldSceneState holds only simulation data for a WorldScene. Logic updates it in Update(dt) from
// intent events; View (Draw) only reads it to present. No rendering or input types.
type WorldSceneState struct {
	World            *object.Manager
	DebugDrawPhysics bool
	EventsLastFrame  uint64 // bus.TakeEventCount() as of the start of the last Update; see drawDebugStats
}

// WorldScene is the engine's generic, data-driven scene type (title + click me button + a
// data-driven GameObject world, scripting, physics, and an optional follow camera). It has no
// knowledge of any specific game's rules — a game that needs its own gameplay logic (hit
// detection, custom spawn conventions, ...) embeds *WorldScene in its own ports.Scene
// implementation instead of adding that logic here (see games/metalslug_demo's Scene, and
// docs/frame_engine_migration_plan.md's "blocking design problem" for why this split exists).
// Logic (Update) runs script components (shared engine per scene), then physics, then world update.
type WorldScene struct {
	titleImg            *ebiten.Image
	uiFace              font.Face
	world               *object.Manager
	engine              script.Engine
	loadedScripts       map[string]bool // path -> true once a load has been attempted (success or failure)
	scriptLoadFailed    map[string]bool // path -> true if that load attempt failed; skips CallScriptUpdate
	scriptUpdateFailing map[string]bool // path -> true while CallScriptUpdate is erroring; logs only on transition
	gameRoot            string          // base path for script loading on OS filesystem (e.g. "games/demo1")
	fsys                fs.FS           // when non-nil, scripts and scenes are loaded from this FS instead
	physicsSystem       *PhysicsSystem
	processes           *process.Manager // timed behaviors with no owning GameObject, e.g. camera shake
	bus                 *event.Bus       // kept only to sample TakeEventCount() for the debug stats overlay
	lastEventCount      uint64           // bus.TakeEventCount() as of the start of the last Update; shown by drawDebugStats
	debugDrawPhysics    bool
	cam                 *camera.Camera // nil unless cfg.Camera.Follow is set
	camTarget           string         // name of the GameObject the camera follows
	worldBuffer         *ebiten.Image  // offscreen sized to the level; drawn to screen translated by -cam.X/Y
	levelWidth          float64        // world bounds for e.g. projectile off-level deactivation; always set, camera or not
	levelHeight         float64
	spawnCount          int // fallback unique-name counter for spawnEntity when payload omits "name"
}

// NewWorldScene returns a new, empty WorldScene.
func NewWorldScene() *WorldScene {
	return &WorldScene{}
}

// Setup loads assets and builds the UI. Implements ports.Scene.
// If config has scene_path set, the world is built from that YAML; otherwise an empty world is used.
// ctx.Bus is used to emit intents (e.g. SceneChangeRequested) and to subscribe to events (e.g. DebugOverlayToggled).
// The script engine backend (Lua or Python) is selected from cfg.ScriptEngine.
func (m *WorldScene) Setup(ctx *ports.SceneContext) error {
	cfg, loader, root, bus := ctx.Config, ctx.Loader, ctx.UI, ctx.Bus
	m.bus = bus

	event.Subscribe(bus, func(ev events.DebugOverlayToggled) {
		m.debugDrawPhysics = !m.debugDrawPhysics
	})
	event.Subscribe(bus, func(ev events.MoveRequested) {
		if controlled := m.FindControlled(); controlled != nil {
			if c := controlled.GetComponent("intent_buffer"); c != nil {
				ib := c.(*object.IntentBuffer)
				ib.PendingMoveX, ib.PendingMoveY = ev.DirX, ev.DirY
			}
		}
	})
	event.Subscribe(bus, func(ev events.ScriptEmitted) {
		if ev.Name == "SpawnEntity" {
			m.spawnEntity(ev.Payload)
		}
		if ev.Name == "ShakeCamera" {
			m.shakeCamera(ev.Payload)
		}
		if m.engine != nil {
			if err := m.engine.CallOnEvent(ev.Name, ev.Payload); err != nil {
				log.Printf("script: on_event(%q) failed: %v", ev.Name, err)
			}
		}
	})
	event.Subscribe(bus, func(ev events.BeginContact) {
		if m.engine != nil {
			if err := m.engine.CallOnEvent("BeginContact", map[string]interface{}{
				"GameObjectNameA": ev.GameObjectNameA,
				"GameObjectNameB": ev.GameObjectNameB,
			}); err != nil {
				log.Printf("script: on_event(\"BeginContact\") failed: %v", err)
			}
		}
	})
	event.Subscribe(bus, func(ev events.EndContact) {
		if m.engine != nil {
			if err := m.engine.CallOnEvent("EndContact", map[string]interface{}{
				"GameObjectNameA": ev.GameObjectNameA,
				"GameObjectNameB": ev.GameObjectNameB,
			}); err != nil {
				log.Printf("script: on_event(\"EndContact\") failed: %v", err)
			}
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
	m.scriptLoadFailed = make(map[string]bool)
	m.scriptUpdateFailing = make(map[string]bool)
	m.processes = process.NewManager()

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
	getEntityPosition := func(name, axis string) (float64, bool) {
		if m.world == nil {
			return 0, false
		}
		obj := m.world.Find(name)
		if obj == nil || !obj.Active {
			return 0, false
		}
		t := obj.Transform()
		if t == nil {
			return 0, false
		}
		if axis == "y" {
			return t.Y, true
		}
		return t.X, true
	}
	m.engine.RegisterEngineAPI(playSound, switchScene, quit, emit, getEntityPosition)

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
		m.physicsSystem.SetEventEmitter(bus)
		m.physicsSystem.InitFromWorld(m.world)
		m.physicsSystem.LogBodies()
	}

	// Level bounds: always set (falls back to viewport size when layout.level_width/height are
	// unset), regardless of whether a camera is active. Used by e.g. updateProjectiles to
	// deactivate projectiles that fly past the level edge — a Logic-layer concern independent of
	// whatever the View currently has on screen.
	m.levelWidth, m.levelHeight = float64(cfg.Layout.LevelWidth), float64(cfg.Layout.LevelHeight)
	if m.levelWidth <= 0 {
		m.levelWidth = float64(cfg.Layout.Width)
	}
	if m.levelHeight <= 0 {
		m.levelHeight = float64(cfg.Layout.Height)
	}

	// Camera-follow: only set up if the scene opts in via cfg.Camera.Follow.
	if cfg.Camera.Follow != "" {
		m.cam = camera.New(cfg.Layout.Width, cfg.Layout.Height, int(m.levelWidth), int(m.levelHeight))
		m.camTarget = cfg.Camera.Follow
		m.worldBuffer = ebiten.NewImage(int(m.levelWidth), int(m.levelHeight))
	}

	return nil
}

// World returns the scene's GameObject world, or nil before Setup runs. Exported so a game's own
// ports.Scene implementation (embedding *WorldScene) can add its own gameplay rules on top.
func (m *WorldScene) World() *object.Manager {
	return m.world
}

// FindControlled returns the first active GameObject with an intent_buffer component — the single
// player-controlled entity a scene is expected to have (e.g. "knight" in demo1, "player" in
// metalslug_demo). Not name-specific, so scenes are free to name their controlled entity anything.
func (m *WorldScene) FindControlled() *object.GameObject {
	if m.world == nil {
		return nil
	}
	for _, go_ := range m.world.Objects() {
		if go_.Active && !go_.IsPrototype && go_.GetComponent("intent_buffer") != nil {
			return go_
		}
	}
	return nil
}

// cameraTargetCenter returns the world-space center of the camera's follow target, and whether it
// was found. Center is the transform position plus half the physics body size when present,
// falling back to the raw transform position (top-left) otherwise.
func (m *WorldScene) cameraTargetCenter() (x, y float64, ok bool) {
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

// destroyDeactivatedPhysicsBodies permanently removes the physics body of any GameObject a
// script deactivated this frame (self.destroy(), see script/python_engine.go) but that still has
// a live PhysicsBody.Body -- generic cleanup for any deactivated object, not specific to any one
// game's use of it (e.g. the metalslug demo's falling-sphere hazard). Runs right after
// updateScripts and before the physics step, so a body that a script just destroyed is never
// simulated for one extra frame. See PhysicsSystem.DestroyBody for why Active=false alone isn't
// enough for a dynamic body.
func (m *WorldScene) destroyDeactivatedPhysicsBodies() {
	for _, obj := range m.world.Objects() {
		if obj.Active || obj.IsPrototype {
			continue
		}
		if pb := obj.PhysicsBody(); pb != nil && pb.Body != nil {
			m.physicsSystem.DestroyBody(m.world, obj.Name)
		}
	}
}

// Update implements ports.Scene. Runs script components (shared engine), then physics step, sync, world update.
func (m *WorldScene) Update(dt float64) {
	// Sampled first, so it covers every event emitted since the previous frame's sample point --
	// input intents (emitted in application/game.Game.Update before this scene's Update runs),
	// script/physics-contact events from last frame's updateScripts/physics step below, all of it
	// -- read by drawDebugStats via WorldSceneState, one full frame behind like FPS/TPS already are.
	if m.bus != nil {
		m.lastEventCount = m.bus.TakeEventCount()
	}
	if m.world == nil || m.engine == nil {
		return
	}
	m.updateScripts(dt)
	if m.physicsSystem != nil {
		m.destroyDeactivatedPhysicsBodies()
		m.physicsSystem.Step(dt)
		m.physicsSystem.SyncToWorld(m.world)
	}
	m.world.Update(dt)
	m.updateProjectiles(dt)
	if m.cam != nil {
		// Reset before running processes so concurrent CameraShakes add their offsets together
		// this frame instead of overwriting each other (see shakeCamera).
		m.cam.ShakeX, m.cam.ShakeY = 0, 0
	}
	if m.processes != nil {
		m.processes.Update(dt)
	}
	if m.cam != nil {
		if cx, cy, ok := m.cameraTargetCenter(); ok {
			m.cam.Follow(cx, cy)
		}
	}
}

// updateScripts runs the script engine's update(dt) (or Script.UpdateFuncName) for every active
// GameObject with a script component, loading each script file (or FS-embedded source) on first use.
func (m *WorldScene) updateScripts(dt float64) {
	for _, go_ := range m.world.Objects() {
		if !go_.Active || go_.IsPrototype {
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
					loadErr = readErr
				} else {
					loadErr = m.engine.DoString(scriptPath, string(src))
				}
			} else {
				loadErr = m.engine.DoFile(scriptPath)
			}
			// Mark as attempted regardless of outcome: on failure this stops updateScripts from
			// retrying (and re-logging) the same broken load every frame. scriptLoadFailed is what
			// actually gates CallScriptUpdate below.
			m.loadedScripts[s.Path] = true
			if loadErr != nil {
				log.Printf("script: failed to load %q: %v", scriptPath, loadErr)
				m.scriptLoadFailed[s.Path] = true
			}
		}
		if m.scriptLoadFailed[s.Path] {
			continue
		}
		funcName := s.UpdateFuncName
		if funcName == "" {
			funcName = "update"
		}
		if err := m.engine.CallScriptUpdate(scriptPath, funcName, go_, dt); err != nil {
			// Logged only on the success->failure transition (not every frame a script keeps
			// failing) to respect the "no per-frame logging" rule while still surfacing the error.
			if !m.scriptUpdateFailing[s.Path] {
				log.Printf("script: %s(%.4f) failed in %q: %v", funcName, dt, scriptPath, err)
				m.scriptUpdateFailing[s.Path] = true
			}
			continue
		}
		if m.scriptUpdateFailing[s.Path] {
			log.Printf("script: %s recovered in %q", funcName, scriptPath)
			m.scriptUpdateFailing[s.Path] = false
		}
	}
}

// Draw renders the scene from current state and view assets. Implements ports.Scene.
// View only reads state (WorldSceneState) and does not mutate simulation data.
func (m *WorldScene) Draw(screen *ebiten.Image) {
	state := m.state()
	if m.titleImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(20, 20)
		screen.DrawImage(m.titleImg, op)
	}
	if state.World != nil {
		m.drawParallaxLayers(screen)
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
			op.GeoM.Translate(-(m.cam.X + m.cam.ShakeX), -(m.cam.Y + m.cam.ShakeY))
			screen.DrawImage(m.worldBuffer, op)
		} else {
			state.World.Draw(screen)
			if state.DebugDrawPhysics && m.physicsSystem != nil {
				m.physicsSystem.DrawDebug(screen, state.World)
			}
		}
	}
	// Same F3/DebugOverlayToggled flag as the physics wireframes above; drawn directly onto screen
	// (not the world buffer) last, so it stays fixed in the corner on top of everything else
	// regardless of camera/level size -- see drawDebugStats' comment for what it shows and why.
	if state.DebugDrawPhysics {
		drawDebugStats(screen, state.EventsLastFrame)
	}
}

// state returns the current simulation state for the view to read (Logic updates these fields in Update).
func (m *WorldScene) state() WorldSceneState {
	return WorldSceneState{
		World:            m.world,
		DebugDrawPhysics: m.debugDrawPhysics,
		EventsLastFrame:  m.lastEventCount,
	}
}

// UIFace returns the font face for UI labels. Implements ports.Scene.
func (m *WorldScene) UIFace() font.Face {
	return m.uiFace
}
