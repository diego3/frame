package event

import "goengine/events"

// Emitter is the narrow interface for emitting events. Layers can depend on Emitter instead of *Bus.
// *Bus implements Emitter.
type Emitter interface {
	Emit(ev interface{})
}

// IntentEmitter emits only intent events (View/Application → Logic). Use when a layer must not emit state events.
// Implemented by IntentBus; wraps *Bus and forwards only allowed intent types.
type IntentEmitter interface {
	Emitter
}

// StateEmitter emits only state events (Logic/Application → View). Use when a layer must not emit intents.
// Implemented by StateBus; wraps *Bus and forwards only allowed state types.
type StateEmitter interface {
	Emitter
}

// IntentBus wraps a Bus and allows emitting only intent event types. Use for Human View or input adapter.
type IntentBus struct{ bus *Bus }

// NewIntentBus returns an IntentEmitter that forwards valid intent events to bus.
func NewIntentBus(bus *Bus) IntentEmitter {
	return &IntentBus{bus: bus}
}

// Emit forwards the event to the bus only if it is an allowed intent type; otherwise no-op.
func (i *IntentBus) Emit(ev interface{}) {
	if ev == nil {
		return
	}
	switch ev.(type) {
	case events.SceneChangeRequested, events.QuitRequested, events.DebugOverlayToggled,
		events.MoveRequested, events.ScriptEmitted:
		i.bus.Emit(ev)
	}
}

// StateBus wraps a Bus and allows emitting only state event types. Use for Game Logic.
type StateBus struct{ bus *Bus }

// NewStateBus returns a StateEmitter that forwards valid state events to bus.
func NewStateBus(bus *Bus) StateEmitter {
	return &StateBus{bus: bus}
}

// Emit forwards the event to the bus only if it is an allowed state type; otherwise no-op.
func (s *StateBus) Emit(ev interface{}) {
	if ev == nil {
		return
	}
	switch ev.(type) {
	case events.SceneChanged, events.GameObjectCreated, events.GameObjectDestroyed,
		events.GameObjectActivated, events.GameObjectDeactivated, events.ComponentAdded, events.ComponentRemoved,
		events.BeginContact, events.EndContact:
		s.bus.Emit(ev)
	}
}
