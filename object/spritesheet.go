package object

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Spritesheet draws a single frame from a sheet of same-sized frames (e.g. for animation).
// Frames are laid out in row-major order (left to right, then next row).
// Name identifies this animation (e.g. "idle", "run"); use with Animator to switch at runtime.
// Loop: when true (default), animation cycles; when false, it plays once and stops on the last frame (Finished() then returns true).
type Spritesheet struct {
	Name        string        // action/animation name (e.g. "idle", "run")
	Image       *ebiten.Image // full sheet
	FrameWidth  int           // width of one frame
	FrameHeight int           // height of one frame
	Cols        int           // number of columns (0 = derive from image width / FrameWidth)
	Rows        int           // number of rows (0 = derive from image height / FrameHeight)
	FrameIndex  int           // current frame (0-based)
	FPS         float64       // frames per second for animation; 0 = static (no auto-advance)
	Loop        bool          // when false, play once and stop (default true)
	HasTint     bool          // when true, multiply every drawn pixel's RGB by Tint (alpha untouched)
	Tint        color.RGBA    // e.g. {0,0,0,255} recolors the whole sheet to a black silhouette
	accumulator float64       // for FPS-based animation
	finished    bool          // true when a non-looping animation has reached the last frame
}

// Type implements Component. Returns "spritesheet" or "spritesheet:name" for unique keys per animation.
func (s *Spritesheet) Type() string {
	if s.Name == "" {
		return "spritesheet"
	}
	return "spritesheet:" + s.Name
}

// Clone implements Component. The clone shares the same Image (a cached, read-only asset) and
// starts its own independent animation state (FrameIndex, accumulator, finished are copied by
// value, so advancing one instance's animation doesn't affect the other's).
func (s *Spritesheet) Clone() Component {
	clone := *s
	return &clone
}

// NumFrames returns the total number of frames (Cols * Rows, or derived from image size).
func (s *Spritesheet) NumFrames() int {
	cols, rows := s.Cols, s.Rows
	if cols <= 0 || rows <= 0 {
		if s.Image == nil {
			return 0
		}
		b := s.Image.Bounds()
		w, h := b.Dx(), b.Dy()
		if cols <= 0 {
			cols = w / s.FrameWidth
		}
		if rows <= 0 {
			rows = h / s.FrameHeight
		}
	}
	return cols * rows
}

// Update implements Updater. Advances the frame when FPS > 0. When Loop is false, stops on the last frame.
func (s *Spritesheet) Update(dt float64) {
	if s.FPS <= 0 || s.finished {
		return
	}
	n := s.NumFrames()
	if n <= 0 {
		return
	}
	s.accumulator += dt
	frameDuration := 1.0 / s.FPS
	for s.accumulator >= frameDuration {
		s.accumulator -= frameDuration
		s.FrameIndex++
		if s.FrameIndex >= n {
			if s.Loop {
				s.FrameIndex = s.FrameIndex % n
			} else {
				s.FrameIndex = n - 1
				s.finished = true
				return
			}
		}
	}
}

// Finished returns true when a non-looping animation has reached the last frame.
func (s *Spritesheet) Finished() bool {
	return s.finished
}

// Reset restarts the animation from frame 0 and clears the finished state. Call when playing a one-shot animation again.
func (s *Spritesheet) Reset() {
	s.FrameIndex = 0
	s.accumulator = 0
	s.finished = false
}

// Draw implements Drawer. Draws the current frame at the transform position.
func (s *Spritesheet) Draw(screen *ebiten.Image, transform *Transform) {
	if s.Image == nil || transform == nil || s.FrameWidth <= 0 || s.FrameHeight <= 0 {
		return
	}
	n := s.NumFrames()
	if n == 0 {
		return
	}
	idx := s.FrameIndex
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	cols := s.Cols
	if cols <= 0 {
		cols = s.Image.Bounds().Dx() / s.FrameWidth
	}
	col := idx % cols
	row := idx / cols
	r := image.Rect(
		col*s.FrameWidth, row*s.FrameHeight,
		(col+1)*s.FrameWidth, (row+1)*s.FrameHeight,
	)
	sub := s.Image.SubImage(r).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	// Scale first, then Translate: ebiten's GeoM.Scale scales the *entire* current matrix,
	// translation included, so Translate-then-Scale would also scale the position (e.g. a
	// character at X=100 with ScaleX=2 would be drawn at X=200) instead of just resizing/
	// flipping the image in place before it's moved to its world position.
	op.GeoM.Scale(transform.ScaleX, transform.ScaleY)
	// A negative scale flips the frame about its own local origin (0,0), leaving it spanning
	// [-frameSize*|scale|, 0] instead of [0, frameSize*|scale|] -- left unadjusted, the drawn
	// footprint would jump to the opposite side of transform.X/Y depending on facing (e.g. a
	// left-facing character would be drawn entirely to the left of its own physics body instead
	// of overlapping it). Shifting by the scaled frame size keeps the footprint anchored at
	// transform.X/Y consistently regardless of sign, matching how Block already behaves.
	tx, ty := transform.X, transform.Y
	if transform.ScaleX < 0 {
		tx += float64(s.FrameWidth) * -transform.ScaleX
	}
	if transform.ScaleY < 0 {
		ty += float64(s.FrameHeight) * -transform.ScaleY
	}
	op.GeoM.Translate(tx, ty)
	if s.HasTint {
		// ColorScale multiplies each channel of the source pixel; scaling RGB to the tint's
		// values while leaving alpha at its default (1) recolors every opaque pixel to Tint
		// without altering the frame's shape/alpha (e.g. a fire-colored explosion sheet drawn
		// as a black silhouette, per art direction), regardless of the source artwork's colors.
		op.ColorScale.SetR(float32(s.Tint.R) / 255)
		op.ColorScale.SetG(float32(s.Tint.G) / 255)
		op.ColorScale.SetB(float32(s.Tint.B) / 255)
	}
	screen.DrawImage(sub, op)
}
