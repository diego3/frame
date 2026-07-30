package scene

import (
	"fmt"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// drawDebugStats draws a small performance/graphics/CPU readout in the screen's top-right corner.
// Gated by the same debugDrawPhysics flag/F3 toggle as the physics wireframe overlay (see Draw) --
// this is debug-only, drawn directly onto screen (not the world buffer), so it never scrolls with
// the camera and stays fixed in the corner regardless of level size. Uses ebitenutil.DebugPrintAt
// (a fixed bitmap font, no UI face needed) so this has zero dependency on the scene's own font
// loading having succeeded.
//
// Lines shown, and why each earns its frame-budget cost:
//   - FPS/TPS (ebiten.ActualFPS/ActualTPS): measured draw rate vs. measured update rate. If TPS
//     drops below the configured tick rate (60) while FPS stays healthy, the bottleneck is game
//     logic (scripts, physics) -- not rendering; if both drop together, it's rendering.
//   - GFX/GL (ebiten.ReadDebugInfo + openGLVersionString): which backend Ebiten picked and, best
//     effort, the driver's own OpenGL version string -- confirms the app is actually on hardware-
//     accelerated OpenGL/DirectX/Metal rather than a silent software fallback (see
//     openGLVersionString's doc comment for why the version specifically needs shelling out to
//     glxinfo instead of an ebiten API).
//   - GPU mem: ebiten's own estimate of GPU image (texture) memory in use -- catches a leaking or
//     duplicated image load.
//   - Events/frame: bus.TakeEventCount() (see event.Bus), one frame behind -- total event-bus
//     traffic (intents, script/physics events, deferred-queue included), a proxy for how much the
//     event-driven layers are doing per frame independent of script/physics cost.
//   - CPU/RAM/Cores: this process' CPU-time percentage and peak resident memory (see cpuPercent's
//     and peakRSSBytes' platform-specific doc comments for why they're POSIX-only) plus
//     runtime.GOMAXPROCS(0)/runtime.NumCPU() -- cores the Go scheduler will actually spread work
//     across vs. cores the machine has available.
func drawDebugStats(screen *ebiten.Image, eventsLastFrame uint64) {
	var info ebiten.DebugInfo
	ebiten.ReadDebugInfo(&info)

	cpu, cpuOK := cpuPercent()
	cpuStr := "n/a"
	if cpuOK {
		cpuStr = fmt.Sprintf("%.1f%%", cpu)
	}

	ramStr := "n/a"
	if rss, ok := peakRSSBytes(); ok {
		ramStr = formatBytes(rss)
	}

	stats := fmt.Sprintf(
		"FPS: %0.1f\nTPS: %0.1f\nGFX: %s\nGL: %s\nGPU mem: %s\nEvents/frame: %d\nCPU: %s\nRAM (peak): %s\nCores: %d/%d",
		ebiten.ActualFPS(),
		ebiten.ActualTPS(),
		info.GraphicsLibrary,
		truncate(openGLVersionString(), 32),
		formatBytes(info.TotalGPUImageMemoryUsageInBytes),
		eventsLastFrame,
		cpuStr,
		ramStr,
		runtime.GOMAXPROCS(0), runtime.NumCPU(),
	)
	b := screen.Bounds()
	ebitenutil.DebugPrintAt(screen, stats, b.Dx()-260, 8)
}

// formatBytes renders n bytes as a fixed one-decimal megabyte string (e.g. "3.2 MB"). Pulled out
// as a pure function so it's unit-testable without an *ebiten.Image.
func formatBytes(n int64) string {
	const mb = 1024 * 1024
	return fmt.Sprintf("%.1f MB", float64(n)/mb)
}

// truncate shortens s to at most max runes, marking the cut with a trailing "..." (which itself
// counts toward max) -- used to keep a driver-reported string (e.g. openGLVersionString's, which
// can run to 60+ characters with a distro/Mesa build suffix) from blowing out the debug overlay's
// fixed-width corner box. Pure function, unit-testable without an *ebiten.Image.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
