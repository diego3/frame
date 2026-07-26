package process

// Base implements the common Process bookkeeping (state transitions and child-chaining) so
// concrete process types can embed it and implement only Update. The zero value is a valid,
// Running process with no child.
type Base struct {
	state State
	child Process
}

// State returns the process's current lifecycle state.
func (b *Base) State() State { return b.state }

// Succeed ends the process successfully. No-op if it is not currently Running or Paused.
func (b *Base) Succeed() {
	if b.state == Running || b.state == Paused {
		b.state = Succeeded
	}
}

// Fail ends the process unsuccessfully. No-op if it is not currently Running or Paused.
// A Failed process's Child, if any, never runs.
func (b *Base) Fail() {
	if b.state == Running || b.state == Paused {
		b.state = Failed
	}
}

// Abort stops the process early from outside its own Update logic (e.g. the scene that started it
// is torn down). Same effect as Fail; kept separate so callers can tell intent apart in logs/tests.
func (b *Base) Abort() {
	if b.state == Running || b.state == Paused {
		b.state = Aborted
	}
}

// Pause suspends Update calls until Unpause is called. No-op if not currently Running.
func (b *Base) Pause() {
	if b.state == Running {
		b.state = Paused
	}
}

// Unpause resumes Update calls. No-op if not currently Paused.
func (b *Base) Unpause() {
	if b.state == Paused {
		b.state = Running
	}
}

// Child returns the process to attach when this one succeeds, or nil.
func (b *Base) Child() Process { return b.child }

// AttachChild sets the process to attach when this one succeeds, replacing any previous child.
func (b *Base) AttachChild(child Process) { b.child = child }
