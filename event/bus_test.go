package event

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"goengine/events"
)

func TestBus_EmitCallsHandler(t *testing.T) {
	bus := NewBus()

	called := false
	var got events.SceneChangeRequested

	Subscribe(bus, func(ev events.SceneChangeRequested) {
		called = true
		got = ev
	})

	want := events.SceneChangeRequested{SceneID: "main_menu"}
	Emit(bus, want)

	assert.True(t, called, "handler should be called")
	assert.Equal(t, want, got)
}

func TestBus_MultipleHandlersCalledInOrder(t *testing.T) {
	bus := NewBus()

	type testEvent struct {
		value int
	}

	var order []int

	Subscribe(bus, func(ev testEvent) {
		order = append(order, 1)
	})
	Subscribe(bus, func(ev testEvent) {
		order = append(order, 2)
	})

	Emit(bus, testEvent{value: 42})

	if assert.Len(t, order, 2, "expected both handlers to be called") {
		assert.Equal(t, []int{1, 2}, order, "handlers should be called in registration order")
	}
}

func TestBus_SeparateEventTypesAreIndependent(t *testing.T) {
	bus := NewBus()

	type eventA struct{}
	type eventB struct{}

	var gotA, gotB bool

	Subscribe(bus, func(ev eventA) {
		gotA = true
	})
	Subscribe(bus, func(ev eventB) {
		gotB = true
	})

	Emit(bus, eventA{})
	assert.True(t, gotA, "eventA handler should be called")
	assert.False(t, gotB, "eventB handler should not be called when emitting eventA")

	gotA, gotB = false, false

	Emit(bus, eventB{})
	assert.True(t, gotB, "eventB handler should be called")
	assert.False(t, gotA, "eventA handler should not be called when emitting eventB")
}

func TestBus_EmitWithNoHandlersDoesNothing(t *testing.T) {
	bus := NewBus()

	// Should not panic or crash even though there are no handlers.
	Emit(bus, events.SceneChangeRequested{SceneID: "unused"})
	Emit(bus, events.QuitRequested{})
}

func TestBus_DeferredQueueOrdersNestedEmits(t *testing.T) {
	bus := NewBus()

	type eventA struct{}
	type eventB struct{}

	var order []string

	Subscribe(bus, func(ev eventA) {
		order = append(order, "A1")
		Emit(bus, eventB{})
	})
	Subscribe(bus, func(ev eventA) {
		order = append(order, "A2")
	})
	Subscribe(bus, func(ev eventB) {
		order = append(order, "B")
	})

	Emit(bus, eventA{})

	// With deferred queue semantics we expect all handlers for the original A
	// to run before B is delivered.
	assert.Equal(t, []string{"A1", "A2", "B"}, order)
}

type sceneChangeRecorder struct {
	events []events.SceneChangeRequested
}

func (r *sceneChangeRecorder) Handle(ev events.SceneChangeRequested) {
	r.events = append(r.events, ev)
}

func TestBus_MethodSubscriber(t *testing.T) {
	bus := NewBus()
	rec := &sceneChangeRecorder{}

	Subscribe(bus, rec.Handle)

	ev := events.SceneChangeRequested{SceneID: "next_scene"}
	Emit(bus, ev)

	if assert.Len(t, rec.events, 1, "expected method handler to be called once") {
		assert.Equal(t, ev, rec.events[0])
	}
}

// TestBus_ConcurrentEmitAndSubscribe exercises the RWMutex on Bus by emitting events from many
// goroutines while additional subscribers are being added. When run with -race, this should not
// report data races, and the always-present handler should be called once per Emit.
func TestBus_ConcurrentEmitAndSubscribe(t *testing.T) {
	bus := NewBus()

	var count int64

	// Always-present handler; every Emit should trigger this exactly once.
	Subscribe(bus, func(ev events.SceneChangeRequested) {
		atomic.AddInt64(&count, 1)
	})

	const goroutines = 10
	const emitsPerGoroutine = 1000
	var wg sync.WaitGroup

	// Emit from multiple goroutines concurrently.
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < emitsPerGoroutine; j++ {
				Emit(bus, events.SceneChangeRequested{SceneID: "concurrent"})
			}
		}()
	}

	// Concurrently add extra subscribers; they are no-ops for the assertion below but exercise
	// the write path on the RWMutex.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < goroutines*emitsPerGoroutine; i++ {
			Subscribe(bus, func(ev events.SceneChangeRequested) {})
		}
	}()

	wg.Wait()

	// The always-present subscriber must be called once per Emit, regardless of concurrent
	// subscriptions.
	expected := int64(goroutines * emitsPerGoroutine)
	assert.Equal(t, expected, count)
}

