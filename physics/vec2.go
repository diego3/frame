package physics

import "goengine/vec2"

// Vec2 is a 2D vector in game units (e.g. pixels). Y increases downward.
// Alias for vec2.Vector, so physics code gets vector arithmetic (Add, Sub, Normalize, ...) for
// free instead of redefining it; see package vec2.
type Vec2 = vec2.Vector
