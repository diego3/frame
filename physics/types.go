package physics

// BodyType is the kind of physics body.
type BodyType int

const (
	BodyStatic    BodyType = iota // Immovable (e.g. ground, walls)
	BodyKinematic                 // Moved by velocity (e.g. player)
	BodyDynamic                   // Affected by forces and gravity
)

// ShapeKind is the collision shape type.
type ShapeKind int

const (
	ShapeBox    ShapeKind = iota // Axis-aligned box (Width x Height)
	ShapeCircle                  // Circle with Radius (bouncy balls, etc.)
)

// BodyDef describes a body to create. All units in game space (e.g. pixels).
// Position is the center of the body.
// For ShapeBox use Width/Height. For ShapeCircle use Radius (Width/Height ignored for the shape).
// Restitution: 0 = no bounce, 1 = full bounce. Friction: 0 = slippery, typical 0.2–0.5.
// Density is mass per unit area (dynamic bodies); 0 = engine default (1).
type BodyDef struct {
	Position    Vec2
	Width       float64
	Height      float64
	Radius      float64  // for ShapeCircle (game units)
	Type        BodyType
	Shape       ShapeKind
	Density     float64
	Restitution float64 // 0–1, bounce; 0 = default
	Friction    float64 // 0+; 0 = default
}
