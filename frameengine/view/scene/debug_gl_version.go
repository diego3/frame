//go:build !js

package scene

import (
	"os/exec"
	"strings"
	"sync"
)

var glVersionOnce struct {
	sync.Once
	value string
}

// openGLVersionString returns the driver-reported OpenGL version (e.g. "4.6 (Compat Profile) Mesa
// 23.2.1"), or "n/a" if it can't be determined. Ebiten deliberately doesn't expose this -- it
// abstracts over OpenGL/DirectX/Metal/etc (see ebiten.DebugInfo.GraphicsLibrary, already shown by
// drawDebugStats) precisely so callers don't reach into a specific backend's internals -- so this
// shells out to glxinfo (Linux/X11; the common case for this engine's target so far) once and
// caches the result, rather than poking at ebiten's own GL context. Best-effort: if glxinfo isn't
// installed or this isn't a GLX system (e.g. Wayland-only, macOS, Windows), the cached value is
// just "n/a". Run at most once per process (sync.Once) since shelling out is far too slow to do
// every time the debug overlay draws.
func openGLVersionString() string {
	glVersionOnce.Do(func() {
		glVersionOnce.value = "n/a"
		out, err := exec.Command("glxinfo", "-B").Output()
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if v, ok := strings.CutPrefix(line, "OpenGL version string:"); ok {
				glVersionOnce.value = strings.TrimSpace(v)
				return
			}
		}
	})
	return glVersionOnce.value
}
