package object

// Projectile marks a GameObject as a projectile and carries the data a future movement/lifetime
// system will consume. It has no Updater/Drawer of its own yet — spawning (this component existing
// on a GameObject) and movement are deliberately separate steps; see
// docs/game_concept_metal_slug_demo.md build order step 4 for the movement/off-screen-deactivation
// follow-up. Pair with a Transform (position) and a visual component (e.g. Block) on the same
// GameObject.
type Projectile struct {
	VelX, VelY float64 // world units/sec, once movement is implemented
	Damage     float64
}

// Type implements Component.
func (Projectile) Type() string { return "projectile" }

// Clone implements Component.
func (p *Projectile) Clone() Component {
	clone := *p
	return &clone
}
