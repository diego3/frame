package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// fnProcess is a minimal test double: a Process whose Update calls a function. Tests use this to
// exercise Manager without needing a real behavior.
type fnProcess struct {
	Base
	updates  int
	inits    int
	onUpdate func(p *fnProcess, dt float64)
}

func (p *fnProcess) Update(dt float64) {
	p.updates++
	if p.onUpdate != nil {
		p.onUpdate(p, dt)
	}
}

func (p *fnProcess) Init() { p.inits++ }

func TestManager_RunningProcessIsUpdatedAndKept(t *testing.T) {
	m := NewManager()
	p := &fnProcess{}
	m.Attach(p)

	m.Update(0.1)
	m.Update(0.1)

	assert.Equal(t, 2, p.updates)
	assert.Equal(t, 1, m.Count())
	assert.Equal(t, 1, p.inits, "Init should run once, on Attach")
}

func TestManager_SucceededProcessIsRemoved(t *testing.T) {
	m := NewManager()
	p := &fnProcess{onUpdate: func(p *fnProcess, dt float64) { p.Succeed() }}
	m.Attach(p)

	m.Update(0.1)

	assert.Equal(t, Succeeded, p.State())
	assert.Equal(t, 0, m.Count())
}

func TestManager_FailedProcessIsRemoved(t *testing.T) {
	m := NewManager()
	p := &fnProcess{onUpdate: func(p *fnProcess, dt float64) { p.Fail() }}
	m.Attach(p)

	m.Update(0.1)

	assert.Equal(t, Failed, p.State())
	assert.Equal(t, 0, m.Count())
}

func TestManager_AbortedProcessIsRemovedWithoutUpdating(t *testing.T) {
	m := NewManager()
	p := &fnProcess{}
	m.Attach(p)
	p.Abort()

	m.Update(0.1)

	assert.Equal(t, 0, p.updates, "aborted process should not be updated")
	assert.Equal(t, 0, m.Count())
}

func TestManager_PausedProcessIsNotUpdatedButKept(t *testing.T) {
	m := NewManager()
	p := &fnProcess{}
	m.Attach(p)
	p.Pause()

	m.Update(0.1)
	assert.Equal(t, 0, p.updates)
	assert.Equal(t, 1, m.Count(), "paused process should stay attached")

	p.Unpause()
	m.Update(0.1)
	assert.Equal(t, 1, p.updates, "process should resume updating after Unpause")
}

func TestManager_SucceededProcessAttachesChildNextUpdate(t *testing.T) {
	m := NewManager()
	child := &fnProcess{}
	parent := &fnProcess{onUpdate: func(p *fnProcess, dt float64) { p.Succeed() }}
	parent.AttachChild(child)
	m.Attach(parent)

	m.Update(0.1)
	assert.Equal(t, 0, child.updates, "child should not run the same frame the parent succeeds")
	assert.Equal(t, 1, m.Count(), "child should now be the sole attached process")

	m.Update(0.1)
	assert.Equal(t, 1, child.updates)
	assert.Equal(t, 1, child.inits)
}

func TestManager_FailedProcessDropsChild(t *testing.T) {
	m := NewManager()
	child := &fnProcess{}
	parent := &fnProcess{onUpdate: func(p *fnProcess, dt float64) { p.Fail() }}
	parent.AttachChild(child)
	m.Attach(parent)

	m.Update(0.1)

	assert.Equal(t, 0, m.Count(), "child of a failed process should never run")
	assert.Equal(t, 0, child.updates)
}

func TestManager_MultipleProcessesUpdateIndependently(t *testing.T) {
	m := NewManager()
	a := &fnProcess{}
	b := &fnProcess{onUpdate: func(p *fnProcess, dt float64) { p.Succeed() }}
	c := &fnProcess{}
	m.Attach(a)
	m.Attach(b)
	m.Attach(c)

	m.Update(0.1)

	assert.Equal(t, 1, a.updates)
	assert.Equal(t, Succeeded, b.State())
	assert.Equal(t, 1, c.updates)
	assert.Equal(t, 2, m.Count(), "a and c remain; b was removed")
}
