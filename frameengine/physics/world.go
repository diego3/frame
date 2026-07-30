package physics

// Body is a handle to a physics body. Implementations may wrap a library-specific type.
// All methods use game units (e.g. pixels). Y increases downward.
type Body interface {
	// GetPosition returns the body position (center) in game units.
	GetPosition() Vec2
	// GetAngle returns the body rotation in radians.
	GetAngle() float64
	// GetLinearVelocity returns velocity in game units per second (e.g. for gameplay or bounce response).
	GetLinearVelocity() Vec2
	// SetLinearVelocity sets velocity in game units per second. Used for kinematic control or to kick dynamic bodies.
	SetLinearVelocity(v Vec2)
	// SetTransform teleports the body to the given position and angle (radians).
	SetTransform(pos Vec2, angle float64)

	// ApplyForce applies a force at a world point (game units). Force magnitude = acceleration in game units/s² (scaled by body mass internally).
	// Use for continuous thrust, wind, etc. Point is in world (game) coordinates; use body center for linear-only effect.
	ApplyForce(force Vec2, point Vec2)
	// ApplyForceToCenter applies a force at the body center (no torque). Force in game units/s².
	ApplyForceToCenter(force Vec2)
	// ApplyLinearImpulse applies an instant change in velocity at a world point. Impulse = desired delta velocity in game units/s (scaled by mass internally).
	// Use for one-shot kicks, bounces, or explosions.
	ApplyLinearImpulse(impulse Vec2, point Vec2)
	// ApplyLinearImpulseToCenter applies an instant velocity change at the body center (no spin).
	ApplyLinearImpulseToCenter(impulse Vec2)
}

// ContactEvent represents a contact between two GameObjects (for callback purposes).
type ContactEvent struct {
	GameObjectNameA string
	GameObjectNameB string
}

// World runs the physics simulation. Create bodies, step each frame, then read positions back.
// No type from any concrete physics library appears here.
type World interface {
	// CreateBody creates a body from the definition. Returns the body handle or an error.
	CreateBody(def BodyDef) (Body, error)
	// Step advances the simulation by dt (seconds). Call once per frame.
	Step(dt float64)
	// DestroyBody permanently removes body from the simulation (and any name registered for it
	// via RegisterBodyName). After this call, body must not be used again.
	DestroyBody(body Body)
	// RegisterBodyName registers a GameObject name for a body (for contact reporting).
	// Call after creating a body if you want contact events.
	RegisterBodyName(body Body, name string)
	// GetContactsNamesThisFrame returns pairs of GameObject names that began/ended contact this frame.
	// Only works if RegisterBodyName was called for the bodies.
	GetContactsNamesThisFrame() (began, ended [][2]string)
}
