package object

// Transform holds position, rotation, and scale. Most GameObjects have one.
type Transform struct {
	X, Y   float64 // position in logical coordinates
	Angle  float64 // rotation in radians
	ScaleX float64
	ScaleY float64
}

// Type implements Component.
func (Transform) Type() string { return "transform" }

// NewTransform returns a Transform at (x, y) with scale 1.
func NewTransform(x, y float64) *Transform {
	return &Transform{X: x, Y: y, ScaleX: 1, ScaleY: 1}
}
