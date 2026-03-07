package input

import (
	"goengine/event"
)

// Adapter reads keys from a Manager and emits intent events to an IntentEmitter.
// Call Poll each frame before game logic Update so Logic reacts to intents instead of raw input.
// ScriptEvents maps input action names to script event names; when an action is just pressed,
// adapter emits event.ScriptEmitted{Name: eventName}. If nil, no script events are emitted.
type Adapter struct {
	mgr          *Manager
	emit         event.IntentEmitter
	scriptEvents map[string]string // action -> script event name
}

// NewAdapter returns an adapter that uses mgr for key state and emits intents to emit.
// scriptEvents is optional: map input action (e.g. "dash") to script event name (e.g. "DashRequested").
func NewAdapter(mgr *Manager, emit event.IntentEmitter, scriptEvents map[string]string) *Adapter {
	if scriptEvents == nil {
		scriptEvents = make(map[string]string)
	}
	return &Adapter{mgr: mgr, emit: emit, scriptEvents: scriptEvents}
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

	// Data-driven: emit ScriptEmitted for each configured action -> event name
	for action, eventName := range a.scriptEvents {
		if eventName == "" {
			continue
		}
		if a.mgr.ActionJustPressed(action) {
			a.emit.Emit(event.ScriptEmitted{Name: eventName, Payload: nil})
		}
	}
}
