// Package metalslug is the Metal Slug demo's own scene: the engine's generic scene.WorldScene
// toolkit, plus this demo's own gameplay rules that don't belong in a general-purpose engine
// (see docs/frame_engine_migration_plan.md's "blocking design problem" for why this split
// exists, and docs/game_concept_metal_slug_demo.md for the demo's design).
package metalslug

import (
	"math"

	"goengine/event"
	"goengine/events"
	"goengine/object"
	"goengine/ports"
	"goengine/view/scene"
)

// Scene embeds the engine's generic WorldScene and adds exactly two Metal-Slug-specific rules on
// top of it: auto-aimed shooting from the player-controlled entity (spawnProjectile), and
// projectile-vs-enemy hit detection (updateHitDetection). Everything else (Setup boilerplate,
// physics, camera, scripting, the generic spawnEntity mechanism) is inherited from *WorldScene
// unchanged.
type Scene struct {
	*scene.WorldScene
}

// NewScene returns a new Metal Slug demo scene.
func NewScene() *Scene {
	return &Scene{WorldScene: scene.NewWorldScene()}
}

// Setup implements ports.Scene: builds the generic world via WorldScene, then subscribes to this
// demo's own "SpawnProjectile" convention (see spawnProjectile) on top of it.
func (s *Scene) Setup(ctx *ports.SceneContext) error {
	if err := s.WorldScene.Setup(ctx); err != nil {
		return err
	}
	event.Subscribe(ctx.Bus, func(ev events.ScriptEmitted) {
		if ev.Name == "SpawnProjectile" {
			s.spawnProjectile(ev.Payload)
		}
	})
	return nil
}

// Update implements ports.Scene: runs the generic world update, then this demo's own hit
// detection (projectile vs. enemy).
func (s *Scene) Update(dt float64) {
	s.WorldScene.Update(dt)
	s.updateHitDetection()
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
//
// This — the hardcoded "projectile_prototype" name and the "SpawnProjectile" event name it's
// wired to — is exactly the kind of demo-specific convention that has no business in the engine's
// generic scene.WorldScene: a different game is free to name things differently, or not have a
// player-shooting mechanic at all.
func (s *Scene) spawnProjectile(payload map[string]interface{}) {
	world := s.World()
	if world == nil {
		return
	}
	proto := world.Find("projectile_prototype")
	if proto == nil || !proto.IsPrototype {
		return
	}
	origin := s.FindControlled()
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

	dirX, dirY := scene.NormalizeDir(scene.PayloadFloat(payload, "dir_x", 1), scene.PayloadFloat(payload, "dir_y", 0))

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

	world.Add(proj)
}

// updateHitDetection checks every active projectile against every active enemy for an AABB
// (axis-aligned bounding box) overlap — a plain rectangle intersection, not a Box2D contact. This
// is the build plan's explicit recommendation for step 5: start with the simplest possible check
// and only reach for real physics contacts (or a spatial partition, if the naive O(projectiles ×
// enemies) loop ever shows up in profiling) if AABB proves insufficient — see
// docs/game_concept_metal_slug_demo.md. On a hit: the projectile is consumed (deactivated) and the
// enemy takes Projectile.Damage, dying (deactivated) at HP <= 0.
//
// This rule — projectiles damage enemies — is this demo's own gameplay logic, not something a
// general-purpose engine scene should assume every game wants.
func (s *Scene) updateHitDetection() {
	world := s.World()
	if world == nil {
		return
	}
	for _, projGo := range world.Objects() {
		if !projGo.Active || projGo.IsPrototype {
			continue
		}
		proj, ok := projGo.GetComponent("projectile").(*object.Projectile)
		if !ok {
			continue
		}
		px, py, pw, ph, ok := scene.AABB(projGo)
		if !ok {
			continue
		}
		for _, enemyGo := range world.Objects() {
			if !enemyGo.Active || enemyGo.IsPrototype {
				continue
			}
			enemy, ok := enemyGo.GetComponent("enemy").(*object.Enemy)
			if !ok {
				continue
			}
			ex, ey, ew, eh, ok := scene.AABB(enemyGo)
			if !ok || !scene.AABBOverlap(px, py, pw, ph, ex, ey, ew, eh) {
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
