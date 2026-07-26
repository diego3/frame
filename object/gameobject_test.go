package object

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
