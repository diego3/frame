package scene

import (
	"math"

	"goengine/object"
	"goengine/physics"
)

// KnightController drives the "knight" object in the world from intent events (MoveRequested, DashRequested, etc.).
// Pending intents are set by event handlers (e.g. MainMenu subscribes to intents and sets these); Update applies them.
// If the knight has a PhysicsBody, movement is done by setting velocity; otherwise Transform is updated directly.
type KnightController struct {
	DashCooldown    float64
	DashActiveUntil float64
	LastMoveX       float64
	LastMoveY       float64

	// Pending intents (set by intent event handlers, consumed in Update)
	PendingMoveX, PendingMoveY float64
	PendingDash                bool
	PendingAttack              bool
	PendingAttack2             bool
}

const (
	knightSpeed     = 50   // pixels per second
	dashImpulse     = 280  // instant velocity boost in dash direction (game units/s)
	dashCooldownSec = 0.5  // seconds between dashes
	dashDurationSec = 0.12 // seconds to preserve dash velocity before resuming normal movement
	dashDirNoMove   = 1.0  // when no keys held, dash in LastMove; if never moved, dash right (X=1)
)

// Update runs one frame of knight gameplay. Uses pending intents (set by intent handlers) and applies to world.
func (c *KnightController) Update(world *object.World, dt float64) {
	knight := world.Find("knight")
	if knight == nil || knight.Animator() == nil {
		return
	}
	// Consume one-shot pendings this frame
	dash := c.PendingDash
	attack := c.PendingAttack
	attack2 := c.PendingAttack2
	c.PendingDash = false
	c.PendingAttack = false
	c.PendingAttack2 = false

	if c.DashCooldown > 0 {
		c.DashCooldown -= dt
	}
	if c.DashActiveUntil > 0 {
		c.DashActiveUntil -= dt
	}
	// Movement from pending direction (emitted each frame by input adapter while keys held)
	vx := c.PendingMoveX * knightSpeed
	vy := c.PendingMoveY * knightSpeed
	if c.PendingMoveX != 0 || c.PendingMoveY != 0 {
		c.LastMoveX, c.LastMoveY = c.PendingMoveX, c.PendingMoveY
	}
	if pb := knight.PhysicsBody(); pb != nil {
		if dash && c.DashCooldown <= 0 {
			var dx, dy float64
			if vx != 0 || vy != 0 {
				len := vx*vx + vy*vy
				if len > 0 {
					len = 1.0 / math.Sqrt(len)
					dx, dy = vx*len*dashImpulse, vy*len*dashImpulse
				}
			} else {
				dx, dy = c.LastMoveX*dashImpulse, c.LastMoveY*dashImpulse
				if dx == 0 && dy == 0 {
					dx = dashDirNoMove * dashImpulse
				}
			}
			if dx != 0 || dy != 0 {
				pb.Body.SetLinearVelocity(physics.Vec2{X: dx, Y: dy})
				c.DashCooldown = dashCooldownSec
				c.DashActiveUntil = dashDurationSec
				knight.Animator().Play("dash")
				knight.ResetAnimation("dash")
			}
		}
		if c.DashActiveUntil <= 0 {
			pb.Body.SetLinearVelocity(physics.Vec2{X: vx, Y: vy})
		}
	}

	anim := knight.Animator()
	switch {
	case attack:
		anim.Play("attack")
		knight.ResetAnimation("attack")
	case attack2:
		anim.Play("attack2")
		knight.ResetAnimation("attack2")
	}
	// When a one-shot animation (attack, attack2, dash) finishes, return to idle
	if anim.Current == "attack" || anim.Current == "attack2" || anim.Current == "dash" {
		if c := knight.GetComponent("spritesheet:" + anim.Current); c != nil {
			if s, ok := c.(*object.Spritesheet); ok && s.Finished() {
				anim.Play("idle")
			}
		}
	} else {
		if c.PendingMoveX != 0 || c.PendingMoveY != 0 {
			anim.Play("run")
		} else {
			anim.Play("idle")
		}
	}
}
