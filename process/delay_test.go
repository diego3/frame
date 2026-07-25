package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDelay_SucceedsAfterElapsedSeconds(t *testing.T) {
	tests := []struct {
		name    string
		seconds float64
		steps   []float64
		want    State
	}{
		{"not yet elapsed", 1.0, []float64{0.4, 0.4}, Running},
		{"elapsed exactly", 1.0, []float64{0.5, 0.5}, Succeeded},
		{"elapsed over multiple steps", 0.3, []float64{0.1, 0.1, 0.1, 0.1}, Succeeded},
		{"zero delay succeeds on first update", 0, []float64{0.016}, Succeeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDelay(tt.seconds)
			for _, dt := range tt.steps {
				d.Update(dt)
			}
			assert.Equal(t, tt.want, d.State())
		})
	}
}

func TestDelay_ChainsIntoChildViaManager(t *testing.T) {
	m := NewManager()
	child := &fnProcess{}
	delay := NewDelay(0.2)
	delay.AttachChild(child)
	m.Attach(delay)

	m.Update(0.1) // 0.1s elapsed, not done yet
	assert.Equal(t, Running, delay.State())
	assert.Equal(t, 0, child.updates)

	m.Update(0.1) // 0.2s elapsed, delay succeeds this frame
	assert.Equal(t, Succeeded, delay.State())
	assert.Equal(t, 0, child.updates, "child starts next frame, not the frame the delay succeeds")

	m.Update(0.1)
	assert.Equal(t, 1, child.updates)
}
