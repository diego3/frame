package object

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

func TestSprite_Type(t *testing.T) {
	assert.Equal(t, "sprite", (&Sprite{}).Type())
}

func TestSprite_Clone(t *testing.T) {
	src := &Sprite{Image: nil, Layer: 2, RepeatWidth: 240}

	clone := src.Clone().(*Sprite)

	assert.NotSame(t, src, clone)
	assert.Equal(t, src.Image, clone.Image)
	assert.Equal(t, 2, clone.Layer)
	assert.Equal(t, 240.0, clone.RepeatWidth)

	clone.RepeatWidth = 480
	assert.Equal(t, 240.0, src.RepeatWidth, "mutating the clone must not affect the source")
}

func TestSpriteRepeatXs(t *testing.T) {
	tests := []struct {
		name                    string
		tx, scaledWidth, repeat float64
		want                    []float64
	}{
		{"repeat unset draws once at tx", 10, 32, 0, []float64{10}},
		{"non-positive scaled width draws once at tx", 10, 0, 240, []float64{10}},
		{"exact multiple tiles precisely", 0, 32, 96, []float64{0, 32, 64}},
		{"partial final tile is included", 0, 32, 100, []float64{0, 32, 64, 96}},
		{"tiling starts at tx, not zero", 100, 32, 64, []float64{100, 132}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := spriteRepeatXs(tt.tx, tt.scaledWidth, tt.repeat)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSprite_Draw_repeatWidthDoesNotPanic(t *testing.T) {
	s := &Sprite{Image: ebiten.NewImage(8, 8), RepeatWidth: 100}
	tr := &Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}
	screen := ebiten.NewImage(200, 50)

	assert.NotPanics(t, func() { s.Draw(screen, tr) })
}

func TestSprite_Draw_nilImageOrTransformIsNoop(t *testing.T) {
	screen := ebiten.NewImage(10, 10)

	assert.NotPanics(t, func() { (&Sprite{Image: nil}).Draw(screen, &Transform{ScaleX: 1, ScaleY: 1}) })
	assert.NotPanics(t, func() { (&Sprite{Image: ebiten.NewImage(4, 4)}).Draw(screen, nil) })
}
