package camera

import "testing"

func TestNew(t *testing.T) {
	c := New(800, 600, 2400, 600)
	if c.ViewportWidth != 800 || c.ViewportHeight != 600 {
		t.Fatalf("viewport = (%v, %v), want (800, 600)", c.ViewportWidth, c.ViewportHeight)
	}
	if c.LevelWidth != 2400 || c.LevelHeight != 600 {
		t.Fatalf("level = (%v, %v), want (2400, 600)", c.LevelWidth, c.LevelHeight)
	}
	if c.X != 0 || c.Y != 0 {
		t.Fatalf("initial offset = (%v, %v), want (0, 0)", c.X, c.Y)
	}
}

func TestCamera_Follow_centersOnTargetInMiddleOfLevel(t *testing.T) {
	c := New(800, 600, 2400, 600)
	c.Follow(1200, 300)
	if got, want := c.X, 800.0; got != want {
		t.Errorf("X = %v, want %v", got, want)
	}
	if got, want := c.Y, 0.0; got != want {
		t.Errorf("Y = %v, want %v", got, want)
	}
}

func TestCamera_Follow_clampsToZeroAtLeftEdge(t *testing.T) {
	c := New(800, 600, 2400, 600)
	c.Follow(50, 0)
	if got, want := c.X, 0.0; got != want {
		t.Errorf("X = %v, want %v (clamped to level start)", got, want)
	}
}

func TestCamera_Follow_clampsToMaxOffsetAtRightEdge(t *testing.T) {
	c := New(800, 600, 2400, 600)
	c.Follow(2350, 0)
	if got, want := c.X, 1600.0; got != want { // 2400 - 800
		t.Errorf("X = %v, want %v (clamped to level end)", got, want)
	}
}

func TestCamera_Follow_noScrollWhenLevelNotLargerThanViewport(t *testing.T) {
	tests := []struct {
		name             string
		levelW, levelH   int
		targetX, targetY float64
	}{
		{"level equals viewport", 800, 600, 100, 500},
		{"level smaller than viewport", 400, 300, 350, 250},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(800, 600, tt.levelW, tt.levelH)
			c.Follow(tt.targetX, tt.targetY)
			if c.X != 0 || c.Y != 0 {
				t.Errorf("offset = (%v, %v), want (0, 0)", c.X, c.Y)
			}
		})
	}
}

func TestCamera_Follow_independentAxes(t *testing.T) {
	c := New(800, 600, 2400, 1800)
	c.Follow(2350, 50)
	if got, want := c.X, 1600.0; got != want {
		t.Errorf("X = %v, want %v", got, want)
	}
	if got, want := c.Y, 0.0; got != want {
		t.Errorf("Y = %v, want %v", got, want)
	}
}
