package camera

import (
	"math/rand"

	"goengine/frameengine/process"
)

// CameraShake is a process.Process (see the process package's process-manager pattern) that adds
// a decaying random offset to a Camera's ShakeX/ShakeY every frame for Duration seconds, then
// succeeds. It has no owning GameObject — a follow camera isn't one — which is exactly the case
// the process package exists for.
//
// Multiple CameraShakes can run on the same Camera at once (e.g. two explosions close together):
// each Update adds its own contribution rather than overwriting the field, so the caller driving
// the process.Manager (see view/scene.WorldScene.Update) is expected to zero Camera.ShakeX/ShakeY
// once per frame before running it, letting concurrent shakes stack.
type CameraShake struct {
	process.Base
	cam       *Camera
	Duration  float64
	Magnitude float64

	elapsed float64
}

// NewCameraShake returns a CameraShake that perturbs cam's ShakeX/ShakeY for duration seconds,
// starting at magnitude (world units, per axis) and decaying linearly to zero. duration <= 0
// succeeds on the first Update without applying any offset, matching process.Delay's convention.
func NewCameraShake(cam *Camera, duration, magnitude float64) *CameraShake {
	return &CameraShake{cam: cam, Duration: duration, Magnitude: magnitude}
}

// Update implements process.Process.
func (s *CameraShake) Update(dt float64) {
	s.elapsed += dt
	if s.elapsed >= s.Duration {
		s.Succeed()
		return
	}
	decay := 1 - s.elapsed/s.Duration
	s.cam.ShakeX += (rand.Float64()*2 - 1) * s.Magnitude * decay
	s.cam.ShakeY += (rand.Float64()*2 - 1) * s.Magnitude * decay
}
