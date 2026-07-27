package object

// Projectile marks a GameObject as a projectile and carries the data MainMenu's
// spawnProjectile/updateProjectiles (view/scene/mainmenu.go) consume. Pair with a Transform
// (position) and a visual component (e.g. Block) on the same GameObject.
//
// On a "*_prototype" GameObject (see GameObject.Clone), Speed/SpawnClearance/DespawnMargin are
// template config: read by spawnProjectile but never overwritten on clone, so scene authors tune
// them entirely in YAML. VelX/VelY are the exception -- always recomputed at spawn time from
// Speed and the requested direction, since that direction is only known at spawn time. Damage is
// carried through unmodified from the prototype to every clone.
type Projectile struct {
	VelX, VelY float64 // world units/sec; computed at spawn from Speed * direction
	Damage     float64

	Speed          float64 // world units/sec
	SpawnClearance float64 // extra distance beyond the shooter's own edge to spawn at, in the facing direction
	DespawnMargin  float64 // distance past the level bounds before the projectile is deactivated
}

// Type implements Component.
func (Projectile) Type() string { return "projectile" }

// Clone implements Component.
func (p *Projectile) Clone() Component {
	clone := *p
	return &clone
}
