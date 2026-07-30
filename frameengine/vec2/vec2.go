// Package vec2 provides a 2D vector type with the arithmetic (add, subtract, scale, dot product,
// length, normalize, lerp, rotate) that recurs across the engine's game-logic and view code.
// Deliberately 2D-only: nothing in this engine needs a third or fourth component, a matrix, or a
// quaternion, so this stays a plain Vector rather than pulling in a general-purpose (3D/4D) math
// library. See docs/tdr for the comparison against adopting one.
package vec2

import "math"

// Vector is a 2D vector (or point) of float64 components.
type Vector struct {
	X, Y float64
}

// Zero is the zero vector.
var Zero = Vector{}

// New returns Vector{X: x, Y: y}.
func New(x, y float64) Vector { return Vector{X: x, Y: y} }

// Add returns v + other.
func (v Vector) Add(other Vector) Vector {
	return Vector{X: v.X + other.X, Y: v.Y + other.Y}
}

// Sub returns v - other.
func (v Vector) Sub(other Vector) Vector {
	return Vector{X: v.X - other.X, Y: v.Y - other.Y}
}

// Scale returns v scaled by s.
func (v Vector) Scale(s float64) Vector {
	return Vector{X: v.X * s, Y: v.Y * s}
}

// Negate returns -v.
func (v Vector) Negate() Vector {
	return Vector{X: -v.X, Y: -v.Y}
}

// Dot returns the dot product of v and other.
func (v Vector) Dot(other Vector) float64 {
	return v.X*other.X + v.Y*other.Y
}

// LengthSquared returns the squared length of v. Cheaper than Length when only comparing or
// thresholding magnitudes, since it avoids a sqrt.
func (v Vector) LengthSquared() float64 {
	return v.Dot(v)
}

// Length returns the length (magnitude) of v.
func (v Vector) Length() float64 {
	return math.Sqrt(v.LengthSquared())
}

// Normalize returns v scaled to length 1, or Zero if v is the zero vector (rather than dividing
// by zero).
func (v Vector) Normalize() Vector {
	l := v.Length()
	if l == 0 {
		return Zero
	}
	return v.Scale(1 / l)
}

// Distance returns the distance between v and other.
func (v Vector) Distance(other Vector) float64 {
	return v.Sub(other).Length()
}

// Lerp returns the linear interpolation between v and other at t (t=0 -> v, t=1 -> other); t is
// not clamped, so callers wanting to extrapolate may pass t outside [0, 1].
func (v Vector) Lerp(other Vector, t float64) Vector {
	return v.Add(other.Sub(v).Scale(t))
}

// Rotate returns v rotated by angle radians, counter-clockwise in a standard Y-up frame. Callers
// working in a Y-down frame (e.g. physics.Vec2) will see the visually opposite rotation direction
// for a given positive angle; the underlying math is unaffected by that convention.
func (v Vector) Rotate(angle float64) Vector {
	sin, cos := math.Sincos(angle)
	return Vector{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
	}
}
