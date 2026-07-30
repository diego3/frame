package process

// Delay is a Process that succeeds after Seconds have elapsed. Useful standalone as a one-shot
// timer, or chained via AttachChild to sequence steps (e.g. "wait, then do X").
type Delay struct {
	Base
	Seconds float64

	elapsed float64
}

// NewDelay returns a Delay that Succeeds once seconds have elapsed. Update is a no-op once that
// happens, so seconds <= 0 succeeds on the first Update.
func NewDelay(seconds float64) *Delay {
	return &Delay{Seconds: seconds}
}

// Update implements Process.
func (d *Delay) Update(dt float64) {
	d.elapsed += dt
	if d.elapsed >= d.Seconds {
		d.Succeed()
	}
}
