package scene

import (
	"fmt"
	"image/color"
	"io/fs"
	"math"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

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

// WorldSceneState holds only simulation data for a WorldScene. Logic updates it in Update(dt) from
// intent events; View (Draw) only reads it to present. No rendering or input types.
type WorldSceneState struct {
	World            *object.Manager
	DebugDrawPhysics bool
}

// WorldScene is the engine's generic, data-driven scene type (title + click me button + a
// data-driven GameObject world, scripting, physics, and an optional follow camera). It has no
// knowledge of any specific game's rules — a game that needs its own gameplay logic (hit
// detection, custom spawn conventions, ...) embeds *WorldScene in its own ports.Scene
// implementation instead of adding that logic here (see games/metalslug_demo's Scene, and
// docs/frame_engine_migration_plan.md's "blocking design problem" for why this split exists).
// Logic (Update) runs script components (shared engine per scene), then physics, then world update.
type WorldScene struct {
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
	levelWidth       float64        // world bounds for e.g. projectile off-level deactivation; always set, camera or not
	levelHeight      float64
	spawnCount       int // fallback unique-name counter for spawnEntity when payload omits "name"
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
		if m.engine != nil {
			_ = m.engine.CallOnEvent(ev.Name, ev.Payload)
		}
	})
	event.Subscribe(bus, func(ev events.BeginContact) {
		if m.engine != nil {
			_ = m.engine.CallOnEvent("BeginContact", map[string]interface{}{
				"GameObjectNameA": ev.GameObjectNameA,
				"GameObjectNameB": ev.GameObjectNameB,
			})
		}
	})
	event.Subscribe(bus, func(ev events.EndContact) {
		if m.engine != nil {
			_ = m.engine.CallOnEvent("EndContact", map[string]interface{}{
				"GameObjectNameA": ev.GameObjectNameA,
				"GameObjectNameB": ev.GameObjectNameB,
			})
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

// updateProjectiles moves each active projectile's Transform by its velocity, then deactivates it
// once it leaves the level bounds (by more than its own configurable DespawnMargin -- see
// object.Projectile). This isn't Projectile.Update(dt) because Updater components don't receive
// their sibling Transform (see object.Updater) — moving a projectile needs both, the same reason
// updateScripts and PhysicsSystem.SyncToWorld are also WorldScene-level steps rather than generic
// per-component Update calls. Uses level bounds (m.levelWidth/levelHeight), not the camera
// viewport — projectile lifetime is a Logic concern, independent of what the View currently has
// on screen.
func (m *WorldScene) updateProjectiles(dt float64) {
	for _, go_ := range m.world.Objects() {
		if !go_.Active || go_.IsPrototype {
			continue
		}
		proj, ok := go_.GetComponent("projectile").(*object.Projectile)
		if !ok {
			continue
		}
		t := go_.Transform()
		if t == nil {
			continue
		}
		t.X += proj.VelX * dt
		t.Y += proj.VelY * dt
		margin := proj.DespawnMargin
		if t.X < -margin || t.X > m.levelWidth+margin ||
			t.Y < -margin || t.Y > m.levelHeight+margin {
			go_.Active = false
		}
	}
}

// spawnEntity clones a named prototype GameObject (the Prototype pattern) at the position given
// in payload, and registers a physics body for it if it has one. Triggered by a script calling
// engine.emit("SpawnEntity", {...}) -- this is deliberately the *only* thing WorldScene knows how
// to do generically: deciding when, what, and with what parameters to spawn is a game-rule
// concern that belongs in a script (e.g. games/metalslug_demo/scripts/python/game_manager.py,
// which periodically spawns "sphere_prototype" as a falling-hazard rule specific to that demo),
// not in this scene type, which is also used by other, unrelated games/scenes. Silently does
// nothing if "prototype" is missing/unknown.
//
// Recognized payload keys:
//
//	"prototype"     (string, required) name of a GameObject with prototype: true in the scene
//	"name"          (string, optional) name for the new instance; auto-generated from the
//	                prototype name + a counter if omitted
//	"x", "y"        (number, optional) Transform position override
//	"timer_seconds" (number, optional) overrides a cloned Timer component's Remaining, if present
//	"vel_x", "vel_y" (number, optional) initial linear velocity set on the clone's physics body
//	                (e.g. a lobbed-in-a-parabola bomb), applied once the body is created below;
//	                no-op if the prototype has no physics_body
func (m *WorldScene) spawnEntity(payload map[string]interface{}) {
	if m.world == nil {
		return
	}
	protoName, _ := payload["prototype"].(string)
	if protoName == "" {
		return
	}
	proto := m.world.Find(protoName)
	if proto == nil || !proto.IsPrototype {
		return
	}

	name, _ := payload["name"].(string)
	if name == "" {
		m.spawnCount++
		name = fmt.Sprintf("%s_%d", protoName, m.spawnCount)
	}

	clone := proto.Clone(name)
	if t := clone.Transform(); t != nil {
		if _, ok := payload["x"]; ok {
			t.X = PayloadFloat(payload, "x", t.X)
		}
		if _, ok := payload["y"]; ok {
			t.Y = PayloadFloat(payload, "y", t.Y)
		}
	}
	if _, ok := payload["timer_seconds"]; ok {
		if timer, ok := clone.GetComponent("timer").(*object.Timer); ok {
			timer.Remaining = PayloadFloat(payload, "timer_seconds", timer.Remaining)
		}
	}

	m.world.Add(clone)
	if m.physicsSystem != nil {
		// Safe to call repeatedly: InitFromWorld only creates bodies for objects whose
		// PhysicsBody.Body is still nil, so this only ever creates the one new clone's body.
		m.physicsSystem.InitFromWorld(m.world)
	}
	_, hasVelX := payload["vel_x"]
	_, hasVelY := payload["vel_y"]
	if hasVelX || hasVelY {
		if pb := clone.PhysicsBody(); pb != nil && pb.Body != nil {
			vx := PayloadFloat(payload, "vel_x", 0)
			vy := PayloadFloat(payload, "vel_y", 0)
			pb.Body.SetLinearVelocity(physics.Vec2{X: vx, Y: vy})
		}
	}
}

// AABB returns the world-space bounding box (x, y, w, h) for go_, using its Block size if present
// (both a projectile and this engine's Block placeholders are Blocks), falling back to its
// PhysicsBody size. ok is false if go_ has no Transform or no usable size. Exported so a game's
// own gameplay rules (e.g. hit detection) can reuse this instead of re-deriving bounding boxes.
func AABB(go_ *object.GameObject) (x, y, w, h float64, ok bool) {
	t := go_.Transform()
	if t == nil {
		return 0, 0, 0, 0, false
	}
	if blk, isBlock := go_.GetComponent("block").(*object.Block); isBlock && blk.Width > 0 && blk.Height > 0 {
		return t.X, t.Y, blk.Width, blk.Height, true
	}
	if pb := go_.PhysicsBody(); pb != nil && pb.Width > 0 && pb.Height > 0 {
		return t.X, t.Y, pb.Width, pb.Height, true
	}
	return 0, 0, 0, 0, false
}

// AABBOverlap reports whether two axis-aligned rectangles (top-left x/y, width w, height h) intersect.
func AABBOverlap(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	return x1 < x2+w2 && x1+w1 > x2 && y1 < y2+h2 && y1+h1 > y2
}

// PayloadFloat reads key from payload as a float64, accepting whatever numeric type the script
// engine produced (Lua and Python payload numbers may arrive as float64, int, or int64), or
// fallback if key is absent or not a number.
func PayloadFloat(payload map[string]interface{}, key string, fallback float64) float64 {
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

// NormalizeDir returns (x, y) scaled to unit length, or (1, 0) if both are zero.
func NormalizeDir(x, y float64) (float64, float64) {
	length := math.Sqrt(x*x + y*y)
	if length == 0 {
		return 1, 0
	}
	return x / length, y / length
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
// View only reads state (WorldSceneState) and does not mutate simulation data.
func (m *WorldScene) Draw(screen *ebiten.Image) {
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
func (m *WorldScene) state() WorldSceneState {
	return WorldSceneState{
		World:            m.world,
		DebugDrawPhysics: m.debugDrawPhysics,
	}
}

// UIFace returns the font face for UI labels. Implements ports.Scene.
func (m *WorldScene) UIFace() font.Face {
	return m.uiFace
}
