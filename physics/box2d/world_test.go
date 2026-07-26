package box2d

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goengine/physics"
)

// closeEnough absorbs the float32 round-trip precision loss inherent in box2d-go's Vec2 (float32).
const epsilon = 0.05

func TestNewWorld_nonPositivePixelScale_usesDefault(t *testing.T) {
	tests := []struct {
		name       string
		pixelScale float64
	}{
		{"zero", 0},
		{"negative", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWorld(physics.Vec2{}, tt.pixelScale).(*worldImpl)
			assert.Equal(t, defaultPixelScale, w.scale)
		})
	}
}

func TestNewWorld_positivePixelScale_isKept(t *testing.T) {
	w := NewWorld(physics.Vec2{}, 32).(*worldImpl)
	assert.Equal(t, float64(32), w.scale)
}

func TestCreateBody_positionRoundTripsThroughPixelScale(t *testing.T) {
	w := NewWorld(physics.Vec2{}, 64)

	body, err := w.CreateBody(physics.BodyDef{
		Position: physics.Vec2{X: 128, Y: 256},
		Width:    50,
		Height:   80,
		Type:     physics.BodyStatic,
		Shape:    physics.ShapeBox,
	})
	require.NoError(t, err)

	pos := body.GetPosition()
	assert.InDelta(t, 128, pos.X, epsilon*64)
	assert.InDelta(t, 256, pos.Y, epsilon*64)
}

func TestCreateBody_circleShape(t *testing.T) {
	w := NewWorld(physics.Vec2{}, 64)

	body, err := w.CreateBody(physics.BodyDef{
		Position: physics.Vec2{X: 10, Y: 20},
		Radius:   24,
		Type:     physics.BodyDynamic,
		Shape:    physics.ShapeCircle,
	})

	require.NoError(t, err)
	require.NotNil(t, body)
}

func TestStep_dynamicBodyFallsUnderGravity(t *testing.T) {
	// Y-down game coordinates: gravity Y positive means "down", so a dynamic body should gain
	// positive Y velocity (and move to larger Y) after stepping.
	w := NewWorld(physics.Vec2{X: 0, Y: 800}, 64)

	body, err := w.CreateBody(physics.BodyDef{
		Position: physics.Vec2{X: 0, Y: 0},
		Width:    50,
		Height:   50,
		Type:     physics.BodyDynamic,
		Shape:    physics.ShapeBox,
	})
	require.NoError(t, err)

	startY := body.GetPosition().Y
	for i := 0; i < 30; i++ {
		w.Step(1.0 / 60.0)
	}

	assert.Greater(t, body.GetPosition().Y, startY, "dynamic body should fall (increasing Y) under positive gravity_y")
	assert.Greater(t, body.GetLinearVelocity().Y, 0.0, "dynamic body should gain downward (positive Y) velocity")
}

func TestStep_staticBodyDoesNotMoveUnderGravity(t *testing.T) {
	w := NewWorld(physics.Vec2{X: 0, Y: 800}, 64)

	body, err := w.CreateBody(physics.BodyDef{
		Position: physics.Vec2{X: 0, Y: 0},
		Width:    800,
		Height:   50,
		Type:     physics.BodyStatic,
		Shape:    physics.ShapeBox,
	})
	require.NoError(t, err)

	for i := 0; i < 60; i++ {
		w.Step(1.0 / 60.0)
	}

	pos := body.GetPosition()
	assert.InDelta(t, 0, pos.X, epsilon*64)
	assert.InDelta(t, 0, pos.Y, epsilon*64)
}

func TestSetLinearVelocity_roundTrips(t *testing.T) {
	w := NewWorld(physics.Vec2{}, 64)

	body, err := w.CreateBody(physics.BodyDef{
		Position: physics.Vec2{X: 0, Y: 0},
		Width:    50,
		Height:   50,
		Type:     physics.BodyKinematic,
		Shape:    physics.ShapeBox,
	})
	require.NoError(t, err)

	body.SetLinearVelocity(physics.Vec2{X: 100, Y: -50})

	v := body.GetLinearVelocity()
	assert.InDelta(t, 100, v.X, epsilon*64)
	assert.InDelta(t, -50, v.Y, epsilon*64)
}

func TestSetTransform_movesBody(t *testing.T) {
	w := NewWorld(physics.Vec2{}, 64)

	body, err := w.CreateBody(physics.BodyDef{
		Position: physics.Vec2{X: 0, Y: 0},
		Width:    50,
		Height:   50,
		Type:     physics.BodyKinematic,
		Shape:    physics.ShapeBox,
	})
	require.NoError(t, err)

	body.SetTransform(physics.Vec2{X: 300, Y: 150}, math.Pi/2)

	pos := body.GetPosition()
	assert.InDelta(t, 300, pos.X, epsilon*64)
	assert.InDelta(t, 150, pos.Y, epsilon*64)
	assert.InDelta(t, math.Pi/2, body.GetAngle(), 0.01)
}

func TestApplyLinearImpulseToCenter_changesVelocityInImpulseDirection(t *testing.T) {
	w := NewWorld(physics.Vec2{}, 64)

	body, err := w.CreateBody(physics.BodyDef{
		Position: physics.Vec2{X: 0, Y: 0},
		Width:    50,
		Height:   50,
		Type:     physics.BodyDynamic,
		Shape:    physics.ShapeBox,
		Density:  1,
	})
	require.NoError(t, err)

	before := body.GetLinearVelocity()
	require.InDelta(t, 0, before.X, epsilon*64)

	body.ApplyLinearImpulseToCenter(physics.Vec2{X: 500, Y: 0})

	after := body.GetLinearVelocity()
	assert.Greater(t, after.X, before.X, "impulse along +X should increase X velocity")
}
