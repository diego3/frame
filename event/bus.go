package event

import (
	"reflect"
	"sync"
)

// Bus is a synchronous event bus. Handlers are invoked when Emit is called, in registration order,
// before Emit returns. No goroutines or channels are used. Events emitted from within handlers are
// queued and delivered after the current event finishes delivery.
type Bus struct {
	mu         sync.RWMutex
	handlers   map[reflect.Type][]func(interface{})
	delivering bool
	queue      []interface{}
}

// NewBus returns a new event bus.
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[reflect.Type][]func(interface{})),
	}
}

// Subscribe registers handler to be called whenever an event of type T is emitted on b.
// The handler receives the emitted value as T. Panics if the emitted value is not of type T.
func Subscribe[T any](b *Bus, handler func(T)) {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	wrapper := func(ev interface{}) {
		handler(ev.(T))
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[typ] = append(b.handlers[typ], wrapper)
}

// Emit delivers ev to all handlers registered for that event type on b. Delivery is synchronous;
// handlers run in registration order before Emit returns. If handlers emit further events, those
// are queued and processed after the current event finishes, avoiding deep recursion.
func Emit(b *Bus, ev interface{}) {
	if ev == nil {
		return
	}
	b.emit(ev)
}

// Emit delivers the event on this bus. Bus implements Emitter so layers can depend on the narrow interface.
func (b *Bus) Emit(ev interface{}) {
	b.emit(ev)
}

func (b *Bus) emit(ev interface{}) {
	b.mu.Lock()
	if !b.delivering {
		// Start a new delivery run.
		b.delivering = true
		b.mu.Unlock()
		b.processQueue(ev)
		b.mu.Lock()
		b.delivering = false
		b.mu.Unlock()
		return
	}
	// Already delivering: enqueue for later.
	b.queue = append(b.queue, ev)
	b.mu.Unlock()
}

// processQueue delivers first and any events that are queued while delivering. It runs until there
// are no more events in the queue.
func (b *Bus) processQueue(first interface{}) {
	current := []interface{}{first}
	for len(current) > 0 {
		ev := current[0]
		current = current[1:]
		b.deliver(ev)

		// Drain any events that were enqueued during delivery of ev.
		b.mu.Lock()
		if len(b.queue) > 0 {
			current = append(current, b.queue...)
			b.queue = nil
		}
		b.mu.Unlock()
	}
}

// deliver sends a single event to all handlers registered for its concrete type.
func (b *Bus) deliver(ev interface{}) {
	if ev == nil {
		return
	}
	typ := reflect.TypeOf(ev)
	b.mu.RLock()
	list := b.handlers[typ]
	if len(list) == 0 {
		b.mu.RUnlock()
		return
	}
	// Copy slice so we don't hold the lock during handler execution (avoids deadlock if a handler subscribes or emits).
	handlers := make([]func(interface{}), len(list))
	copy(handlers, list)
	b.mu.RUnlock()
	for _, h := range handlers {
		h(ev)
	}
}
