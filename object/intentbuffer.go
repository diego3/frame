package object

// IntentBuffer holds pending input intents for one frame. Event handlers (e.g. in the scene)
// set these from the bus; script components read them via the entity API (self.get_intent).
// Clear one-shot fields (Dash, Attack, Attack2) after the script update for that object.
type IntentBuffer struct {
	PendingMoveX, PendingMoveY float64
	PendingDash               bool
	PendingAttack             bool
	PendingAttack2            bool
}

// Type implements Component.
func (IntentBuffer) Type() string { return "intent_buffer" }

// ClearOneShot clears one-shot intents. Call after running the entity's script update.
func (ib *IntentBuffer) ClearOneShot() {
	ib.PendingDash = false
	ib.PendingAttack = false
	ib.PendingAttack2 = false
}
