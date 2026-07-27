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
	levelWidth       float64        // world bounds for e.g. projectile off-level deactivation; always set, camera or not
	levelHeight      float64
	spawnCount       int // fallback unique-name counter for spawnEntity when payload omits "name"
}

// NewMainMenu returns a new main menu scene.
func NewMainMenu() *MainMenu {
	return &MainMenu{}
}

// Setup loads assets and builds the UI. Implements ports.Scene.
// If config has scene_path set, the world is built from that YAML; otherwise an empty world is used.
// ctx.Bus is used to emit intents (e.g. SceneChangeRequested) and to subscribe to events (e.g. DebugOverlayToggled).
// The script engine backend (Lua or Python) is selected from cfg.ScriptEngine.
func (m *MainMenu) Setup(ctx *ports.SceneContext) error {
	cfg, loader, root, bus := ctx.Config, ctx.Loader, ctx.UI, ctx.Bus

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

// findControlled returns the first active GameObject with an intent_buffer component — the single
// player-controlled entity a scene is expected to have (e.g. "knight" in demo1, "player" in
// metalslug_demo). Not name-specific, so scenes are free to name their controlled entity anything.
func (m *MainMenu) findControlled() *object.GameObject {
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

// spawnProjectile clones the scene's "projectile_prototype" GameObject (a Prototype-pattern
// template: an inert GameObject marked IsPrototype, defined in the scene YAML like any other
// object, never itself run or drawn — see object.GameObject.Clone and Manager.Update/Draw), then
// repositions and aims the clone at the controlled entity's center, offset in the facing direction
// given by payload's "dir_x"/"dir_y" (defaults to facing right). If the scene has no
// "projectile_prototype", shooting is a silent no-op — a scene that never wires up shooting
// doesn't need one.
//
// Speed/Damage/SpawnClearance/DespawnMargin all come from the prototype's own Projectile
// component (YAML-configurable, see object.Projectile) rather than being hardcoded here, and the
// spawn offset is computed from the *shooter's* own PhysicsBody size (half-extent along whichever
// axis the shot fires on) plus that clearance — not a single fixed distance — so the projectile
// spawns clear of the shooter's body regardless of how big or small that shooter is.
func (m *MainMenu) spawnProjectile(payload map[string]interface{}) {
	if m.world == nil {
		return
	}
	proto := m.world.Find("projectile_prototype")
	if proto == nil || !proto.IsPrototype {
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
	originHalfWidth, originHalfHeight := 0.0, 0.0
	if pb := origin.PhysicsBody(); pb != nil {
		cx += pb.Width / 2
		cy += pb.Height / 2
		originHalfWidth, originHalfHeight = pb.Width/2, pb.Height/2
	}

	dirX, dirY := normalizeDir(payloadFloat(payload, "dir_x", 1), payloadFloat(payload, "dir_y", 0))

	proj := proto.Clone("projectile")

	originHalfExtent := originHalfWidth
	if math.Abs(dirY) > math.Abs(dirX) {
		originHalfExtent = originHalfHeight
	}
	spawnOffset := originHalfExtent
	if p, ok := proj.GetComponent("projectile").(*object.Projectile); ok {
		spawnOffset += p.SpawnClearance
		p.VelX, p.VelY = dirX*p.Speed, dirY*p.Speed
	}

	// Center the projectile on the spawn point using its own visual size (Block), if present.
	halfW, halfH := 0.0, 0.0
	if blk, ok := proj.GetComponent("block").(*object.Block); ok {
		halfW, halfH = blk.Width/2, blk.Height/2
	}
	proj.Transform().X = cx + dirX*spawnOffset - halfW
	proj.Transform().Y = cy + dirY*spawnOffset - halfH

	m.world.Add(proj)
}

// updateProjectiles moves each active projectile's Transform by its velocity, then deactivates it
// once it leaves the level bounds (by more than its own configurable DespawnMargin -- see
// object.Projectile). This isn't Projectile.Update(dt) because Updater components don't receive
// their sibling Transform (see object.Updater) — moving a projectile needs both, the same reason
// updateScripts and PhysicsSystem.SyncToWorld are also MainMenu-level steps rather than generic
// per-component Update calls. Uses level bounds (m.levelWidth/levelHeight), not the camera
// viewport — projectile lifetime is a Logic concern, independent of what the View currently has
// on screen.
func (m *MainMenu) updateProjectiles(dt float64) {
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

// spawnEntity clones a named prototype GameObject (Prototype pattern, the same mechanism
// spawnProjectile already uses) at the position given in payload, and registers a physics body
// for it if it has one. Triggered by a script calling engine.emit("SpawnEntity", {...}) -- this
// is deliberately the *only* thing MainMenu knows how to do generically: deciding when, what, and
// with what parameters to spawn is a game-rule concern that belongs in a script (e.g.
// games/metalslug_demo/scripts/python/game_manager.py, which periodically spawns
// "sphere_prototype" as a falling-hazard rule specific to this demo), not in this scene type,
// which is also used by other, unrelated games/scenes. Silently does nothing if "prototype" is
// missing/unknown, same convention as spawnProjectile.
//
// Recognized payload keys:
//
//	"prototype"     (string, required) name of a GameObject with prototype: true in the scene
//	"name"          (string, optional) name for the new instance; auto-generated from the
//	                prototype name + a counter if omitted
//	"x", "y"        (number, optional) Transform position override
//	"timer_seconds" (number, optional) overrides a cloned Timer component's Remaining, if present
func (m *MainMenu) spawnEntity(payload map[string]interface{}) {
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
			t.X = payloadFloat(payload, "x", t.X)
		}
		if _, ok := payload["y"]; ok {
			t.Y = payloadFloat(payload, "y", t.Y)
		}
	}
	if _, ok := payload["timer_seconds"]; ok {
		if timer, ok := clone.GetComponent("timer").(*object.Timer); ok {
			timer.Remaining = payloadFloat(payload, "timer_seconds", timer.Remaining)
		}
	}

	m.world.Add(clone)
	if m.physicsSystem != nil {
		// Safe to call repeatedly: InitFromWorld only creates bodies for objects whose
		// PhysicsBody.Body is still nil, so this only ever creates the one new clone's body.
		m.physicsSystem.InitFromWorld(m.world)
	}
}

// updateHitDetection checks every active projectile against every active enemy for an AABB
// (axis-aligned bounding box) overlap — a plain rectangle intersection, not a Box2D contact. This
// is the build plan's explicit recommendation for step 5: start with the simplest possible check
// and only reach for real physics contacts (or a spatial partition, if the naive O(projectiles ×
// enemies) loop ever shows up in profiling) if AABB proves insufficient — see
// docs/game_concept_metal_slug_demo.md. On a hit: the projectile is consumed (deactivated) and the
// enemy takes Projectile.Damage, dying (deactivated) at HP <= 0.
func (m *MainMenu) updateHitDetection() {
	for _, projGo := range m.world.Objects() {
		if !projGo.Active || projGo.IsPrototype {
			continue
		}
		proj, ok := projGo.GetComponent("projectile").(*object.Projectile)
		if !ok {
			continue
		}
		px, py, pw, ph, ok := aabb(projGo)
		if !ok {
			continue
		}
		for _, enemyGo := range m.world.Objects() {
			if !enemyGo.Active || enemyGo.IsPrototype {
				continue
			}
			enemy, ok := enemyGo.GetComponent("enemy").(*object.Enemy)
			if !ok {
				continue
			}
			ex, ey, ew, eh, ok := aabb(enemyGo)
			if !ok || !aabbOverlap(px, py, pw, ph, ex, ey, ew, eh) {
				continue
			}
			projGo.Active = false
			enemy.HP -= proj.Damage
			if enemy.HP <= 0 {
				enemyGo.Active = false
			}
			break // this projectile is consumed; stop checking it against other enemies
		}
	}
}

// aabb returns the world-space bounding box (x, y, w, h) for go_, using its Block size if present
// (both the projectile and this demo's enemy/player placeholders are Blocks), falling back to its
// PhysicsBody size. ok is false if go_ has no Transform or no usable size.
func aabb(go_ *object.GameObject) (x, y, w, h float64, ok bool) {
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

// aabbOverlap reports whether two axis-aligned rectangles (top-left x/y, width w, height h) intersect.
func aabbOverlap(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	return x1 < x2+w2 && x1+w1 > x2 && y1 < y2+h2 && y1+h1 > y2
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

// destroyDeactivatedPhysicsBodies permanently removes the physics body of any GameObject a
// script deactivated this frame (self.destroy(), see script/python_engine.go) but that still has
// a live PhysicsBody.Body -- generic cleanup for any deactivated object, not specific to any one
// game's use of it (e.g. the metalslug demo's falling-sphere hazard). Runs right after
// updateScripts and before the physics step, so a body that a script just destroyed is never
// simulated for one extra frame. See PhysicsSystem.DestroyBody for why Active=false alone isn't
// enough for a dynamic body.
func (m *MainMenu) destroyDeactivatedPhysicsBodies() {
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
func (m *MainMenu) Update(dt float64) {
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
	m.updateHitDetection()
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
