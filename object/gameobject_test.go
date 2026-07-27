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

func TestGameObject_Clone(t *testing.T) {
	src := NewGameObject("prototype")
	src.IsPrototype = true
	src.AddComponent(&Transform{X: 10, Y: 20, ScaleX: 1, ScaleY: 1})
	src.AddComponent(&Block{Width: 8, Height: 8})
	src.AddComponent(&Projectile{VelX: 1, VelY: 2, Damage: 3})

	clone := src.Clone("projectile")

	assert.Equal(t, "projectile", clone.Name)
	assert.NotEqual(t, src.ID, clone.ID, "clone should get a fresh ID")
	assert.True(t, clone.Active, "clone should always be Active regardless of the source")
	assert.False(t, clone.IsPrototype, "clone should never itself be a prototype")

	t.Run("component values are copied", func(t *testing.T) {
		ct := clone.Transform()
		if assert.NotNil(t, ct) {
			assert.Equal(t, 10.0, ct.X)
			assert.Equal(t, 20.0, ct.Y)
		}
		cp, ok := clone.GetComponent("projectile").(*Projectile)
		if assert.True(t, ok) {
			assert.Equal(t, Projectile{VelX: 1, VelY: 2, Damage: 3}, *cp)
		}
	})

	t.Run("mutating the clone does not affect the source", func(t *testing.T) {
		clone.Transform().X = 999
		cp := clone.GetComponent("projectile").(*Projectile)
		cp.VelX = 999

		assert.Equal(t, 10.0, src.Transform().X, "source Transform.X should be unaffected")
		srcProj := src.GetComponent("projectile").(*Projectile)
		assert.Equal(t, 1.0, srcProj.VelX, "source Projectile.VelX should be unaffected")
	})

	t.Run("component pointers are distinct", func(t *testing.T) {
		assert.NotSame(t, src.GetComponent("transform"), clone.GetComponent("transform"))
		assert.NotSame(t, src.GetComponent("block"), clone.GetComponent("block"))
		assert.NotSame(t, src.GetComponent("projectile"), clone.GetComponent("projectile"))
	})
}

func TestGameObject_Clone_SpriteSharesImage(t *testing.T) {
	// Sprite/Spritesheet hold a *ebiten.Image, a cached read-only asset that clones should share,
	// not duplicate. A nil Image is enough to verify the pointer itself is copied identically.
	src := NewGameObject("proto")
	sprite := &Sprite{Image: nil, Layer: 2}
	src.AddComponent(sprite)

	clone := src.Clone("instance")

	cloneSprite, ok := clone.GetComponent("sprite").(*Sprite)
	if assert.True(t, ok) {
		assert.Equal(t, sprite.Image, cloneSprite.Image, "clone should share the same Image reference")
		assert.Equal(t, 2, cloneSprite.Layer)
	}
}
