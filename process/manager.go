package process

// Manager owns and updates a set of attached Processes each frame, removing ones that finished
// (Succeeded, Failed, or Aborted) and attaching each succeeded process's Child, if any. Not safe
// for concurrent use — call from the same goroutine that owns the game loop (matches how
// object.Manager and PhysicsSystem are used from Scene.Update).
type Manager struct {
	processes []Process
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{}
}

// Attach adds proc to the manager and runs its Init (if it implements Initer). proc receives its
// first Update on the next call to Manager.Update.
func (m *Manager) Attach(proc Process) {
	if initer, ok := proc.(Initer); ok {
		initer.Init()
	}
	m.processes = append(m.processes, proc)
}

// Count returns the number of currently attached (not yet finished) processes.
func (m *Manager) Count() int {
	return len(m.processes)
}

// Update advances every Running process by dt, leaves Paused ones untouched, and removes any that
// are Done afterward. A process that Succeeded has its Child (if any) attached for the next frame;
// a process that Failed or was Aborted is simply dropped, and its Child (if any) never runs.
func (m *Manager) Update(dt float64) {
	var toAttach []Process
	active := m.processes[:0]
	for _, p := range m.processes {
		if p.State() == Running {
			p.Update(dt)
		}
		switch p.State() {
		case Succeeded:
			if child := p.Child(); child != nil {
				toAttach = append(toAttach, child)
			}
			continue
		case Failed, Aborted:
			continue
		}
		active = append(active, p)
	}
	m.processes = active
	for _, child := range toAttach {
		m.Attach(child)
	}
}
