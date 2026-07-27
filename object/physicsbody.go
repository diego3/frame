package object

import "goengine/physics"

// PhysicsBody holds a physics body (abstraction; no dependency on Box2D or any concrete engine).
// When loaded from YAML, Body is nil until the physics system creates it (InitFromWorld).
// Width and Height are the body size in game units (for box); body center = transform top-left + (w/2, h/2) + (OffsetX, OffsetY).
// For ShapeCircle, Radius is used and Width/Height can be 2*Radius for sync/draw.
// OffsetX, OffsetY shift the body relative to the sprite.
// Restitution: 0 = no bounce, 1 = full bounce. Friction: 0+ (e.g. 0.2–0.5).
type PhysicsBody struct {
	Body        physics.Body
	BodyType    physics.BodyType
	Shape       physics.ShapeKind
	Width       float64
	Height      float64
	Radius      float64
	OffsetX     float64
	OffsetY     float64
	Density     float64
	Mass        float64
	Restitution float64
	Friction    float64
}

// Type implements Component.
func (PhysicsBody) Type() string { return "physics_body" }

// Clone implements Component. The clone shares the same Body reference as the original (a live
// Box2D body cannot be duplicated by value copy); callers cloning a GameObject with a physics_body
// need the physics system to (re)initialize a real Body for the clone before relying on physics.
// No current prototype (e.g. games/metalslug_demo's projectile_prototype) has a physics_body, so
// this doesn't come up in practice yet.
func (pb *PhysicsBody) Clone() Component {
	clone := *pb
	return &clone
}
