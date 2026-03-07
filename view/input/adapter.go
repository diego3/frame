package input

import (
	"goengine/event"
)

// Adapter reads keys from a Manager and emits intent events to an IntentEmitter.
// Call Poll each frame before game logic Update so Logic reacts to intents instead of raw input.
type Adapter struct {
	mgr *Manager
	emit event.IntentEmitter
}

// NewAdapter returns an adapter that uses mgr for key state and emits intents to emit.
func NewAdapter(mgr *Manager, emit event.IntentEmitter) *Adapter {
	return &Adapter{mgr: mgr, emit: emit}
}

// Poll reads current input and emits intent events. Call once per frame from Application/Game Update.
func (a *Adapter) Poll() {
	// Global / app-level intents
	if a.mgr.ActionJustPressed("debug_overlay") {
		a.emit.Emit(event.DebugOverlayToggled{})
	}

	// Movement: emit each frame with current direction (Logic subscribes and applies)
	var dirX, dirY float64
	if a.mgr.ActionPressed("move_left") {
		dirX = -1
	}
	if a.mgr.ActionPressed("move_right") {
		dirX = 1
	}
	if a.mgr.ActionPressed("move_up") {
		dirY = -1
	}
	if a.mgr.ActionPressed("move_down") {
		dirY = 1
	}
	a.emit.Emit(event.MoveRequested{DirX: dirX, DirY: dirY})

	// Knight/demo-specific actions as script events (scripts listen via on_event)
	if a.mgr.ActionJustPressed("dash") {
		a.emit.Emit(event.ScriptEmitted{Name: "DashRequested", Payload: nil})
	}
	if a.mgr.ActionJustPressed("attack") {
		a.emit.Emit(event.ScriptEmitted{Name: "AttackRequested", Payload: nil})
	}
	if a.mgr.ActionJustPressed("attack2") {
		a.emit.Emit(event.ScriptEmitted{Name: "Attack2Requested", Payload: nil})
	}
}
