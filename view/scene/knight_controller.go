package scene

import (
	"math"

	"goengine/physics"
	"goengine/view/input"
	"goengine/object"
)

// KnightController drives the "knight" object in the world: WASD movement (run when moving left/right, idle otherwise),
// dash (Shift), attack/attack2 (J/K), and returning to idle when a one-shot attack finishes.
// If the knight has a PhysicsBody, movement is done by setting velocity; otherwise Transform is updated directly.
type KnightController struct {
	DashCooldown    float64 // seconds until next dash allowed
	DashActiveUntil float64 // while > 0 we don't overwrite velocity (so dash impulse is visible)
	LastMoveX       float64 // last non-zero move direction for dash when not moving
	LastMoveY       float64
}

const (
	knightSpeed     = 50   // pixels per second
	dashImpulse     = 280  // instant velocity boost in dash direction (game units/s)
	dashCooldownSec = 0.5  // seconds between dashes
	dashDurationSec = 0.12 // seconds to preserve dash velocity before resuming normal movement
	dashDirNoMove   = 1.0  // when no keys held, dash in LastMove; if never moved, dash right (X=1)
)

// Update runs one frame of knight gameplay. Moves by WASD, dash on Shift, handles animator input.
func (c *KnightController) Update(world *object.World, dt float64) {
	knight := world.Find("knight")
	if knight == nil || knight.Animator() == nil {
		return
	}
	if c.DashCooldown > 0 {
		c.DashCooldown -= dt
	}
	if c.DashActiveUntil > 0 {
		c.DashActiveUntil -= dt
	}
	// Movement: use physics body velocity if present, else direct transform
	if pb := knight.PhysicsBody(); pb != nil {
		var vx, vy float64
		if input.ActionPressed("move_left") {
			vx = -knightSpeed
			c.LastMoveX, c.LastMoveY = -1, 0
		}
		if input.ActionPressed("move_right") {
			vx = knightSpeed
			c.LastMoveX, c.LastMoveY = 1, 0
		}
		if input.ActionPressed("move_up") {
			vy = -knightSpeed
			if vx == 0 {
				c.LastMoveX, c.LastMoveY = 0, -1
			} else {
				c.LastMoveY = -1
			}
		}
		if input.ActionPressed("move_down") {
			vy = knightSpeed
			if vx == 0 {
				c.LastMoveX, c.LastMoveY = 0, 1
			} else {
				c.LastMoveY = 1
			}
		}
		// Dash: set velocity in current or last move direction (kinematic bodies ignore impulses)
		if input.ActionJustPressed("dash") && c.DashCooldown <= 0 {
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
		// Don't overwrite velocity during dash window so dash is visible
		if c.DashActiveUntil <= 0 {
			pb.Body.SetLinearVelocity(physics.Vec2{X: vx, Y: vy})
		}
	}

	anim := knight.Animator()
	// Attack input (one-shot)
	switch {
	case input.ActionJustPressed("attack"):
		anim.Play("attack")
		knight.ResetAnimation("attack")
	case input.ActionJustPressed("attack2"):
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
		// Movement drives run vs idle: run when moving left or right, idle otherwise
		if input.ActionPressed("move_left") || input.ActionPressed("move_right") {
			anim.Play("run")
		} else {
			anim.Play("idle")
		}
	}
}
