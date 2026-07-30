# Pattern catalog: Game Coding Complete → goengine

Deeper reference for `game-architecture`'s SKILL.md. Read this when the summary there isn't
enough — a chapter-by-chapter mapping, and one full worked example (camera shake, the current
`process.Manager` first-use candidate).

## Chapter mapping

| Game Coding Complete (4th ed.) concept | This codebase | Status |
|---|---|---|
| Ch. 3-4: Game loop, "master game loop" separated from view | `application/game/game.go` (ebiten `Update`/`Draw`), layers per ADR-003 | Implemented |
| Ch. 4: Application layer / game logic / game view split | `application/`, logic in scenes + scripts, `view/` | Implemented (this is ADR-003) |
| Ch. 5: Game actors, "big dumb object"/component composition | `object.GameObject` + `map[string]Component` | Implemented |
| Ch. 5: Actor factory / prototype spawning | `GameObject.IsPrototype` + `Clone()`, `spawnEntity` | Implemented |
| Ch. 6: Resource cache | `resource.Manager` (image/audio/font caching by path) | Implemented |
| Ch. 4: Event/messaging system decoupling subsystems | `event.Bus`, `event.Subscribe`/`Emit`, deferred queue | Implemented |
| Ch. 4: Process manager for time-sliced behavior | `process.Manager`/`Process`/`Base`/`Delay` | Package exists, **not yet attached to any scene** |
| Ch. 12: AI — finite state machines for actor behavior | `enemy_bomber.py`'s idle/attack animation-driven state | Implemented informally, one enemy only |
| Ch. 12: AI — pathfinding, steering | — | Not applicable yet (enemies are stationary or fixed-velocity walkers) |
| Ch. 14: Scripting (embedding a scripting language) | `script/` (Lua via gopher-lua, Python via gpython) | Implemented |
| Ch. 9-10: 3D graphics pipeline, shaders | Ebiten's Kage shader support (2D-only equivalent) | Documented, unused — see `docs/tdr/TDR_008_shader_visual_effects_support.md` |
| Ch. 13: Audio system | `resource.Manager.LoadAudio`/`NewAudioPlayer` | Implemented (basic playback, no mixing/ducking) |
| HUD / game screens as a stack of views | `view/ui.Element` (`Container`, `Button`) | Implemented for menus; no in-game HUD yet |
| Memory management, object pools | Go's GC + Prototype-clone (not a true pool — see below) | Partial |

Chapters not covered here (networking, save games, localization, full 3D rendering) don't
currently apply to this engine's scope; skip them rather than force-fitting a pattern.

### A note on "object pooling"

Game Coding Complete's actor factory and classic object-pool advice both aim at the same
problem — don't pay allocation + GC cost for entities that come and go constantly (bullets,
particles). This engine's `GameObject.Clone()` solves the *"don't hand-assemble a GameObject from
scratch every spawn"* half of that (uniform shape, centralized definition in YAML), but it does
**not** solve the *"don't allocate at all"* half — `Clone()` still allocates a new `GameObject`
and a full copy of its components every call. That's an accepted tradeoff at this engine's scale
(see CLAUDE.md's "minimize allocations" guidance — it says minimize, not eliminate) and matches
how `object.Manager` has no free-list/recycling mechanism today. If frame-time profiling ever
shows spawn-heavy scenes (e.g. many simultaneous spheres/explosions) causing GC pressure, the
next step is a true pool — a `[]*GameObject` of pre-cloned, deactivated instances reused via
`Active = true` instead of `Clone()` — but don't build that preemptively; it's real added
complexity (deactivated-instance bookkeeping) that Game Coding Complete itself only recommends
once profiling shows the need.

## Worked example: CameraShake as a Process

This is the concrete shape for the first thing that should attach to a `process.Manager` — no
scene does this yet, so this is a proposal to implement, not a description of existing code.

```go
package view // or a dedicated package, e.g. view/camera

import "goengine/process"

// CameraShake is a Process that offsets a Camera by a decaying random amount for its duration,
// then restores it to zero. Embeds process.Base for State/Succeed/Fail/Child for free.
type CameraShake struct {
	process.Base

	camera   *camera.Camera
	duration float64
	strength float64 // max offset in pixels at t=0, decaying linearly to 0

	elapsed float64
}

func NewCameraShake(cam *camera.Camera, duration, strength float64) *CameraShake {
	return &CameraShake{camera: cam, duration: duration, strength: strength}
}

func (s *CameraShake) Update(dt float64) {
	s.elapsed += dt
	if s.elapsed >= s.duration {
		s.camera.ShakeX, s.camera.ShakeY = 0, 0
		s.Succeed()
		return
	}
	remaining := 1 - s.elapsed/s.duration // 1 -> 0 over the shake's life
	mag := s.strength * remaining
	s.camera.ShakeX = randRange(-mag, mag)
	s.camera.ShakeY = randRange(-mag, mag)
}
```

Notes on why it's shaped this way:

- **Embeds `process.Base`, not a bespoke struct** — free `State()`/`Succeed()`/`Child()` wiring,
  consistent with every other `Process` implementation (see `process/delay.go` for the minimal
  example).
- **Owns a reference to the thing it affects (`*camera.Camera`), not a copy** — the Process
  *is* the timed mutation of that state; there is deliberately no separate "shake state" living
  on `Camera` itself beyond the `ShakeX`/`ShakeY` fields it writes each frame.
  `Camera.Follow(...)` (existing) should add `ShakeX`/`ShakeY` on top of its clamped offset when
  computing the final draw translation.
- **Resets to zero on completion** — a Process that mutates external state must clean up after
  itself in the same branch where it calls `Succeed()`; leaving `ShakeX`/`ShakeY` at their last
  random value would leave the camera permanently offset.
- **Triggering it**: wherever the "start a shake" decision is made (e.g. `updateHitDetection`
  when an explosion damages the player, or a `ScriptEmitted` handler for a script-driven
  explosion), call `m.processes.Attach(NewCameraShake(m.camera, 0.3, 8))`. Deciding *when* to
  shake is a scene/gameplay-rule decision (matches how `game_manager.py` decides *when* to spawn,
  per its own header comment) — the `CameraShake` Process itself only knows *how*.

This same shape — embed `Base`, own a reference to whatever it mutates, clean up on `Succeed`
— is the template for any future scene-owned timed effect (screen flash, slow-motion, a
scripted cutscene camera pan) that doesn't belong to a single scripted `GameObject`.
