package camera

import (
	"math"
	"testing"

	"goengine/frameengine/process"
)

func TestCameraShake_RunningUntilDurationElapses(t *testing.T) {
	c := &Camera{}
	s := NewCameraShake(c, 1.0, 8)

	s.Update(0.4)
	if s.State() != process.Running {
		t.Fatalf("state = %v, want Running before duration elapses", s.State())
	}

	s.Update(0.4)
	if s.State() != process.Running {
		t.Fatalf("state = %v, want Running (0.8s of 1.0s elapsed)", s.State())
	}

	s.Update(0.2)
	if s.State() != process.Succeeded {
		t.Fatalf("state = %v, want Succeeded once duration elapses", s.State())
	}
}

func TestCameraShake_ZeroDurationSucceedsImmediatelyWithoutOffset(t *testing.T) {
	c := &Camera{}
	s := NewCameraShake(c, 0, 8)

	s.Update(0.016)

	if s.State() != process.Succeeded {
		t.Fatalf("state = %v, want Succeeded", s.State())
	}
	if c.ShakeX != 0 || c.ShakeY != 0 {
		t.Fatalf("shake offset = (%v, %v), want (0, 0) for a zero-duration shake", c.ShakeX, c.ShakeY)
	}
}

func TestCameraShake_OffsetNeverExceedsMagnitude(t *testing.T) {
	c := &Camera{}
	s := NewCameraShake(c, 1.0, 8)

	s.Update(0.001) // near the start: decay is close to 1, the widest the offset can be

	if math.Abs(c.ShakeX) > 8 || math.Abs(c.ShakeY) > 8 {
		t.Fatalf("shake offset = (%v, %v), want magnitude within +/-8", c.ShakeX, c.ShakeY)
	}
}

func TestCameraShake_OffsetBoundShrinksAsDurationElapses(t *testing.T) {
	c := &Camera{}
	s := NewCameraShake(c, 1.0, 8)

	s.Update(0.99) // 1% of the duration remains: decay ~= 0.01, so the bound is ~0.08

	const bound = 8 * 0.02 // generous slack around the 0.01 decay factor
	if math.Abs(c.ShakeX) > bound || math.Abs(c.ShakeY) > bound {
		t.Fatalf("shake offset = (%v, %v), want within +/-%v near the end of the shake", c.ShakeX, c.ShakeY, bound)
	}
}
