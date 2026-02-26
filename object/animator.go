package object

// Animator holds the current animation name. When present, World.Draw only draws
// the spritesheet component with key "spritesheet:"+Current (so one animation is visible at a time).
type Animator struct {
	Current string // active animation name (e.g. "idle", "run")
}

// Type implements Component.
func (Animator) Type() string { return "animator" }

// Play sets the current animation by name.
func (a *Animator) Play(name string) {
	a.Current = name
}

// Toggle switches between two animations: if current is a, switch to b; otherwise switch to a.
func (a *Animator) Toggle(aName, bName string) {
	if a.Current == aName {
		a.Current = bName
	} else {
		a.Current = aName
	}
}
