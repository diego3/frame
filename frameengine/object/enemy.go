package object

// Enemy marks a GameObject as an enemy with hit points, taking damage from projectile contact
// (see view/scene/mainmenu.go's updateHitDetection, a simple AABB overlap check — no Box2D
// contacts; see docs/game_concept_metal_slug_demo.md). Movement is script-driven (a "script"
// component walking at a constant velocity), same pattern as the player controller — Enemy itself
// carries no velocity/speed field, matching how PLAYER_SPEED lives in the script, not a component.
type Enemy struct {
	HP float64
}

// Type implements Component.
func (Enemy) Type() string { return "enemy" }

// Clone implements Component.
func (e *Enemy) Clone() Component {
	clone := *e
	return &clone
}
