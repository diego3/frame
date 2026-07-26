package object

// IntentBuffer holds pending movement input for one frame. Event handlers (e.g. in the scene)
// set PendingMoveX/Y from MoveRequested; script components read them via self.get_intent("move_x"|"move_y").
type IntentBuffer struct {
	PendingMoveX, PendingMoveY float64
}

// Type implements Component.
func (IntentBuffer) Type() string { return "intent_buffer" }

// Clone implements Component.
func (ib *IntentBuffer) Clone() Component {
	clone := *ib
	return &clone
}
