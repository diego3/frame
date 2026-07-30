package object

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParallaxLayer_Type(t *testing.T) {
	assert.Equal(t, "parallax_layer", (&ParallaxLayer{}).Type())
}

func TestParallaxLayer_Clone(t *testing.T) {
	// Image is a cached, read-only asset that clones should share, not duplicate -- same
	// convention as Sprite/Spritesheet.Clone (see TestGameObject_Clone_SpriteSharesImage). A nil
	// Image is enough to verify the pointer itself is copied identically.
	src := &ParallaxLayer{Image: nil, ScrollFactor: 0.3, Repeat: true}

	clone := src.Clone().(*ParallaxLayer)

	assert.NotSame(t, src, clone)
	assert.Equal(t, src.Image, clone.Image)
	assert.Equal(t, 0.3, clone.ScrollFactor)
	assert.True(t, clone.Repeat)

	// Mutating the clone must not affect the source.
	clone.ScrollFactor = 0.9
	assert.Equal(t, 0.3, src.ScrollFactor)
}
