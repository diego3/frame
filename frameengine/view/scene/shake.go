package scene

import "goengine/frameengine/view/camera"

// shakeCamera starts a camera.CameraShake process in response to a script calling
// engine.emit("ShakeCamera", {"duration": ..., "magnitude": ...}) -- e.g.
// games/metalslug_demo/scripts/python/sphere_timer.py emits this alongside its "SpawnEntity" call
// for the explosion VFX, the moment a sphere's fuse runs out. This is deliberately generic (not
// metalslug-specific): any scene with a follow camera can trigger a shake the same way. Silently
// does nothing if the scene has no camera, same convention as spawnEntity/spawnProjectile.
//
// Recognized payload keys:
//
//	"duration"  (number, optional) seconds the shake lasts; defaults to 0.3
//	"magnitude" (number, optional) peak per-axis offset in world units; defaults to 8
func (m *WorldScene) shakeCamera(payload map[string]interface{}) {
	if m.cam == nil {
		return
	}
	duration := PayloadFloat(payload, "duration", 0.3)
	magnitude := PayloadFloat(payload, "magnitude", 8)
	m.processes.Attach(camera.NewCameraShake(m.cam, duration, magnitude))
}
