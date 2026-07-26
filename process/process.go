// Package process implements a process manager: composable, updatable behaviors that run across
// multiple frames (cooldowns, delays, timed sequences), following the process-manager pattern
// described in "Game Coding Complete" (McShaffry & Graham). A Process runs until it succeeds,
// fails, or is aborted; on success it can hand off to a chained child Process, so multi-step
// sequences (e.g. "wait, then flash, then despawn") compose without a scene or script hand-rolling
// its own timers.
package process

// State is where a Process is in its lifecycle. The zero value is Running, so a freshly
// constructed Base (and any type embedding it) is alive as soon as it is attached.
type State int

const (
	// Running processes receive Update(dt) calls every frame.
	Running State = iota
	// Paused processes are kept alive but do not receive Update(dt) calls until Unpause is called.
	Paused
	// Succeeded processes are done; the Manager removes them and attaches their Child, if any.
	Succeeded
	// Failed processes are done and removed; a Failed process's Child, if any, never runs.
	Failed
	// Aborted processes were stopped early from outside (e.g. Abort); removed the same as Failed.
	Aborted
)

// String returns a short name for s, for logging/debugging.
func (s State) String() string {
	switch s {
	case Running:
		return "running"
	case Paused:
		return "paused"
	case Succeeded:
		return "succeeded"
	case Failed:
		return "failed"
	case Aborted:
		return "aborted"
	default:
		return "unknown"
	}
}

// Done reports whether s is a terminal state (Succeeded, Failed, or Aborted); the Manager removes
// a Process once its State is Done.
func (s State) Done() bool {
	return s == Succeeded || s == Failed || s == Aborted
}

// Process is a single behavior that runs over one or more frames, managed by a Manager.
// Implementations should embed Base to get State/Succeed/Fail/Pause/Unpause/Abort/child-chaining
// for free, and only need to implement Update.
type Process interface {
	// Update advances the process by dt seconds. Implementations end the process by calling
	// Succeed or Fail (usually via an embedded Base); leaving the state as Running continues it
	// next frame. The Manager does not call Update while the process is Paused or already Done.
	Update(dt float64)

	// State returns the process's current lifecycle state.
	State() State

	// Child returns the Process to attach when this one Succeeds, or nil for none.
	Child() Process
}

// Initer is an optional interface a Process can implement to run one-time setup the frame it is
// attached to a Manager, before its first Update. Mirrors the object package's optional
// Updater/Drawer pattern: most processes don't need it and can omit it entirely.
type Initer interface {
	// Init runs once, when the process is attached to a Manager.
	Init()
}
