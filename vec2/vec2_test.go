package vec2

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVector_Add(t *testing.T) {
	got := New(1, 2).Add(New(3, 4))
	assert.Equal(t, New(4, 6), got)
}

func TestVector_Sub(t *testing.T) {
	got := New(3, 4).Sub(New(1, 2))
	assert.Equal(t, New(2, 2), got)
}

func TestVector_Scale(t *testing.T) {
	got := New(2, 3).Scale(2)
	assert.Equal(t, New(4, 6), got)
}

func TestVector_Negate(t *testing.T) {
	got := New(2, -3).Negate()
	assert.Equal(t, New(-2, 3), got)
}

func TestVector_Dot(t *testing.T) {
	got := New(1, 2).Dot(New(3, 4))
	assert.Equal(t, 11.0, got)
}

func TestVector_LengthSquaredAndLength(t *testing.T) {
	v := New(3, 4)
	assert.Equal(t, 25.0, v.LengthSquared())
	assert.Equal(t, 5.0, v.Length())
}

func TestVector_Normalize(t *testing.T) {
	tests := []struct {
		name string
		v    Vector
		want Vector
	}{
		{"unit x-axis", New(5, 0), New(1, 0)},
		{"3-4-5 triangle", New(3, 4), New(0.6, 0.8)},
		{"zero vector stays zero", Zero, Zero},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.Normalize()
			assert.InDelta(t, tt.want.X, got.X, 1e-9)
			assert.InDelta(t, tt.want.Y, got.Y, 1e-9)
		})
	}
}

func TestVector_Distance(t *testing.T) {
	got := New(0, 0).Distance(New(3, 4))
	assert.Equal(t, 5.0, got)
}

func TestVector_Lerp(t *testing.T) {
	tests := []struct {
		name string
		t    float64
		want Vector
	}{
		{"t=0 returns start", 0, New(0, 0)},
		{"t=1 returns end", 1, New(10, 20)},
		{"t=0.5 returns midpoint", 0.5, New(5, 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(0, 0).Lerp(New(10, 20), tt.t)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVector_Rotate(t *testing.T) {
	v := New(1, 0)

	quarter := v.Rotate(math.Pi / 2)
	assert.InDelta(t, 0.0, quarter.X, 1e-9)
	assert.InDelta(t, 1.0, quarter.Y, 1e-9)

	half := v.Rotate(math.Pi)
	assert.InDelta(t, -1.0, half.X, 1e-9)
	assert.InDelta(t, 0.0, half.Y, 1e-9)

	full := v.Rotate(2 * math.Pi)
	assert.InDelta(t, 1.0, full.X, 1e-9)
	assert.InDelta(t, 0.0, full.Y, 1e-9)
}
