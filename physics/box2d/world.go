package box2d

import (
	"math"

	"goengine/physics"

	b2 "github.com/oliverbestmann/box2d-go"
)

// pixelScale converts game units (pixels) to Box2D meters. 1 meter = pixelScale pixels.
// Box2D uses Y-up; we use Y-down, so we flip Y when converting.
const defaultPixelScale = 64.0

// worldImpl implements physics.World using box2d-go.
type worldImpl struct {
	w    b2.World
	scale float64
}

// bodyImpl implements physics.Body.
type bodyImpl struct {
	b     b2.Body
	scale float64
}

// NewWorld creates a physics world using Box2D. Gravity is in game units per second² (Y positive = down).
// pixelScale is how many game units (pixels) equal 1 meter in Box2D (e.g. 64).
func NewWorld(gravity physics.Vec2, pixelScale float64) physics.World {
	if pixelScale <= 0 {
		pixelScale = defaultPixelScale
	}
	def := b2.DefaultWorldDef()
	// Box2D Y-up: our gravity (0, 98) pixels/s² down -> (0, -98/scale) m/s²
	def.Gravity = b2.Vec2{X: float32(gravity.X / pixelScale), Y: float32(-gravity.Y / pixelScale)}
	w := b2.CreateWorld(def)
	return &worldImpl{w: w, scale: pixelScale}
}

func (w *worldImpl) gameToB2(pos physics.Vec2) b2.Vec2 {
	return b2.Vec2{
		X: float32(pos.X / w.scale),
		Y: float32(-pos.Y / w.scale),
	}
}

func (w *worldImpl) b2ToGame(v b2.Vec2) physics.Vec2 {
	return physics.Vec2{
		X: float64(v.X) * w.scale,
		Y: -float64(v.Y) * w.scale,
	}
}

func (w *worldImpl) CreateBody(def physics.BodyDef) (physics.Body, error) {
	bdef := b2.DefaultBodyDef()
	bdef.Position = w.gameToB2(def.Position)
	switch def.Type {
	case physics.BodyStatic:
		bdef.Type1 = b2.StaticBody
	case physics.BodyKinematic:
		bdef.Type1 = b2.KinematicBody
	case physics.BodyDynamic:
		bdef.Type1 = b2.DynamicBody
	default:
		bdef.Type1 = b2.StaticBody
	}
	body := w.w.CreateBody(bdef)
	shapeDef := b2.DefaultShapeDef()
	if def.Density > 0 {
		shapeDef.Density = float32(def.Density)
	} else {
		shapeDef.Density = 1
	}
	if def.Restitution > 0 || def.Friction > 0 {
		shapeDef.Material = b2.DefaultSurfaceMaterial()
		if def.Restitution >= 0 {
			shapeDef.Material.Restitution = float32(def.Restitution)
		}
		if def.Friction >= 0 {
			shapeDef.Material.Friction = float32(def.Friction)
		}
	}
	switch def.Shape {
	case physics.ShapeCircle:
		radius := float32(def.Radius / w.scale)
		if radius <= 0 {
			radius = 0.5
		}
		circle := b2.Circle{Radius: radius}
		body.CreateCircleShape(shapeDef, circle)
	default:
		hw := float32(def.Width/w.scale) * 0.5
		hh := float32(def.Height/w.scale) * 0.5
		if hw <= 0 {
			hw = 0.5
		}
		if hh <= 0 {
			hh = 0.5
		}
		box := b2.MakeBox(hw, hh)
		body.CreatePolygonShape(shapeDef, box)
	}
	return &bodyImpl{b: body, scale: w.scale}, nil
}

func (w *worldImpl) Step(dt float64) {
	// Standard Box2D step: subStepCount 4
	w.w.Step(float32(dt), 4)
}

func (b *bodyImpl) GetPosition() physics.Vec2 {
	pos := b.b.GetPosition()
	return physics.Vec2{X: float64(pos.X) * b.scale, Y: -float64(pos.Y) * b.scale}
}

func (b *bodyImpl) GetAngle() float64 {
	rot := b.b.GetRotation()
	return float64(rot.Angle())
}

func (b *bodyImpl) GetLinearVelocity() physics.Vec2 {
	v := b.b.GetLinearVelocity()
	return physics.Vec2{X: float64(v.X) * b.scale, Y: -float64(v.Y) * b.scale}
}

func (b *bodyImpl) SetLinearVelocity(v physics.Vec2) {
	// Velocity: game units/sec -> m/s
	b.b.SetLinearVelocity(b2.Vec2{
		X: float32(v.X / b.scale),
		Y: float32(-v.Y / b.scale),
	})
}

func (b *bodyImpl) SetTransform(pos physics.Vec2, angle float64) {
	rot := b2.Rot{C: float32(math.Cos(angle)), S: float32(math.Sin(angle))}
	b.b.SetTransform(b2.Vec2{
		X: float32(pos.X / b.scale),
		Y: float32(-pos.Y / b.scale),
	}, rot)
}

func (b *bodyImpl) gameToB2Vec2(v physics.Vec2) b2.Vec2 {
	return b2.Vec2{X: float32(v.X / b.scale), Y: float32(-v.Y / b.scale)}
}

func (b *bodyImpl) ApplyForce(force physics.Vec2, point physics.Vec2) {
	mass := b.b.GetMass()
	// F_b2 = (force_game/scale)*mass so that a_game = force_game
	f := b2.Vec2{
		X: float32(force.X/b.scale) * mass,
		Y: float32(-force.Y/b.scale) * mass,
	}
	b.b.ApplyForce(f, b.gameToB2Vec2(point), 1)
}

func (b *bodyImpl) ApplyForceToCenter(force physics.Vec2) {
	mass := b.b.GetMass()
	f := b2.Vec2{
		X: float32(force.X/b.scale) * mass,
		Y: float32(-force.Y/b.scale) * mass,
	}
	b.b.ApplyForceToCenter(f, 1)
}

func (b *bodyImpl) ApplyLinearImpulse(impulse physics.Vec2, point physics.Vec2) {
	mass := b.b.GetMass()
	// impulse_b2 = mass * (deltaV_game/scale), Y flipped
	imp := b2.Vec2{
		X: float32(impulse.X/b.scale) * mass,
		Y: float32(-impulse.Y/b.scale) * mass,
	}
	b.b.ApplyLinearImpulse(imp, b.gameToB2Vec2(point), 1)
}

func (b *bodyImpl) ApplyLinearImpulseToCenter(impulse physics.Vec2) {
	mass := b.b.GetMass()
	imp := b2.Vec2{
		X: float32(impulse.X/b.scale) * mass,
		Y: float32(-impulse.Y/b.scale) * mass,
	}
	b.b.ApplyLinearImpulseToCenter(imp, 1)
}

