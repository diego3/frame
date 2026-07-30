package scene

import (
	"fmt"
	"math"

	"goengine/frameengine/object"
	"goengine/frameengine/physics"
)

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
