package event

import (
	"reflect"
	"sync"
)

// Bus is a synchronous event bus. Handlers are invoked when Emit is called, in registration order,
// before Emit returns. No goroutines or channels are used.
type Bus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type][]func(interface{})
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

// Emit delivers ev to all handlers registered for that event type. Delivery is synchronous;
// handlers run in registration order before Emit returns.
func (b *Bus) Emit(ev interface{}) {
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
