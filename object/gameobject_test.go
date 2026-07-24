package object

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGameObject(t *testing.T) {
	g := NewGameObject("knight")

	assert.Equal(t, "knight", g.Name)
	assert.True(t, g.Active)
	assert.NotZero(t, g.ID)
}

func TestNewGameObject_assignsUniqueIDs(t *testing.T) {
	a := NewGameObject("a")
	b := NewGameObject("b")

	assert.NotEqual(t, a.ID, b.ID)
}

func TestAddComponent_nilIsNoOp(t *testing.T) {
	g := NewGameObject("obj")

	g.AddComponent(nil)

	assert.Nil(t, g.GetComponent("transform"))
}

func TestAddComponent_duplicateTypeReplaces(t *testing.T) {
	g := NewGameObject("obj")

	g.AddComponent(&Transform{X: 1})
	g.AddComponent(&Transform{X: 2})

	tr, ok := g.GetComponent("transform").(*Transform)
	require.True(t, ok)
	assert.Equal(t, float64(2), tr.X)
}

func TestGetComponent_unknownTypeReturnsNil(t *testing.T) {
	g := NewGameObject("obj")

	assert.Nil(t, g.GetComponent("nope"))
}

func TestTransform_returnsNilWhenAbsent(t *testing.T) {
	g := NewGameObject("obj")
	assert.Nil(t, g.Transform())
}

func TestTransform_returnsComponentWhenPresent(t *testing.T) {
	g := NewGameObject("obj")
	tr := NewTransform(5, 10)
	g.AddComponent(tr)

	assert.Same(t, tr, g.Transform())
}

func TestAnimator_returnsNilWhenAbsent(t *testing.T) {
	g := NewGameObject("obj")
	assert.Nil(t, g.Animator())
}

func TestAnimator_returnsComponentWhenPresent(t *testing.T) {
	g := NewGameObject("obj")
	anim := &Animator{Current: "idle"}
	g.AddComponent(anim)

	assert.Same(t, anim, g.Animator())
}

func TestPhysicsBody_returnsNilWhenAbsent(t *testing.T) {
	g := NewGameObject("obj")
	assert.Nil(t, g.PhysicsBody())
}

func TestPhysicsBody_returnsComponentWhenPresent(t *testing.T) {
	g := NewGameObject("obj")
	pb := &PhysicsBody{Width: 50, Height: 80}
	g.AddComponent(pb)

	assert.Same(t, pb, g.PhysicsBody())
}

// AnimatedComponent centralizes the "only one active animation" rule (moved out of Manager's
// Update/Draw loop). These cases cover every branch: no Animator, empty Current, Current set with
// no matching spritesheet, and Current set with a match.
func TestAnimatedComponent(t *testing.T) {
	t.Run("no animator", func(t *testing.T) {
		g := NewGameObject("obj")

		c, ok := g.AnimatedComponent()

		assert.False(t, ok)
		assert.Nil(t, c)
	})

	t.Run("animator with empty Current", func(t *testing.T) {
		g := NewGameObject("obj")
		g.AddComponent(&Animator{Current: ""})

		c, ok := g.AnimatedComponent()

		assert.False(t, ok)
		assert.Nil(t, c)
	})

	t.Run("animator set but no matching spritesheet", func(t *testing.T) {
		g := NewGameObject("obj")
		g.AddComponent(&Animator{Current: "run"})

		c, ok := g.AnimatedComponent()

		assert.True(t, ok, "caller should still skip iterating all components")
		assert.Nil(t, c)
	})

	t.Run("animator set with matching spritesheet", func(t *testing.T) {
		g := NewGameObject("obj")
		g.AddComponent(&Animator{Current: "run"})
		sheet := &Spritesheet{Name: "run"}
		g.components["spritesheet:run"] = sheet

		c, ok := g.AnimatedComponent()

		assert.True(t, ok)
		assert.Same(t, sheet, c)
	})
}

func TestResetAnimation_noMatchIsNoOp(t *testing.T) {
	g := NewGameObject("obj")

	assert.NotPanics(t, func() { g.ResetAnimation("missing") })
}

func TestResetAnimation_resetsFrameIndex(t *testing.T) {
	g := NewGameObject("obj")
	sheet := &Spritesheet{Name: "run", FrameIndex: 7}
	g.components["spritesheet:run"] = sheet

	g.ResetAnimation("run")

	assert.Equal(t, 0, sheet.FrameIndex)
}
