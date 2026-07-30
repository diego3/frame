// Package camera provides a simple 2D follow camera: a world-space offset clamped to level
// bounds so the viewport never shows space outside the level. It has no rendering dependency —
// callers translate their draw target by (X, Y) themselves (see view/scene.WorldScene.Draw).
package camera

// Camera holds a world-space top-left offset for the visible viewport, plus the viewport and
// level sizes needed to clamp that offset.
type Camera struct {
	X, Y                          float64
	ViewportWidth, ViewportHeight float64
	LevelWidth, LevelHeight       float64
}

// New returns a camera for the given viewport and level size. If the level is no larger than the
// viewport on an axis, the offset on that axis is always clamped to zero (no scroll).
func New(viewportWidth, viewportHeight, levelWidth, levelHeight int) *Camera {
	return &Camera{
		ViewportWidth:  float64(viewportWidth),
		ViewportHeight: float64(viewportHeight),
		LevelWidth:     float64(levelWidth),
		LevelHeight:    float64(levelHeight),
	}
}

// Follow centers the camera on (targetX, targetY) — typically a follow target's center in world
// coordinates — then clamps the offset so the viewport never shows space outside the level.
func (c *Camera) Follow(targetX, targetY float64) {
	c.X = clamp(targetX-c.ViewportWidth/2, 0, maxOffset(c.LevelWidth, c.ViewportWidth))
	c.Y = clamp(targetY-c.ViewportHeight/2, 0, maxOffset(c.LevelHeight, c.ViewportHeight))
}

// maxOffset returns the largest valid top-left offset on an axis given the level and viewport
// size: 0 when the level doesn't exceed the viewport, otherwise the remaining scroll room.
func maxOffset(level, viewport float64) float64 {
	if level <= viewport {
		return 0
	}
	return level - viewport
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
