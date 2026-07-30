package object

// Timer counts down a duration in seconds. It exists as a component (per-instance state
// attached to a GameObject) rather than a Python module-level variable because every
// GameObject that shares a script path shares that script's module-level globals too
// (see script/python_engine.go's PythonEngine.modules, keyed by path, not by GameObject) --
// a plain global would collide across multiple concurrent instances of the same script
// (e.g. several falling spheres exploding on independent timers at once).
type Timer struct {
	Remaining float64 // seconds left; a script decrements this itself via self.get_timer()/set_timer()
}

// Type implements Component.
func (Timer) Type() string { return "timer" }

// Clone implements Component.
func (t *Timer) Clone() Component {
	clone := *t
	return &clone
}
