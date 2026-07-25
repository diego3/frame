# TDR-008: Shader / visual-effects support

## Status

Known

## Context

Ebiten (already pinned at `v2.9.0` in `go.mod`) has built-in GPU shader support via **Kage**, its own Go-like shading language. Kage source compiles at runtime (`ebiten.NewShader`) to whatever backend is active (OpenGL, Metal, DirectX, WebGL), so the engine would never write backend-specific shader code. Shaders are applied via `Image.DrawRectShader` / `Image.DrawTrianglesShader`, which take uniforms (per-draw parameters — time, intensity, a color, etc.) and up to a few bound source images.

None of this is wired into `goengine` today. Every draw path in `object/` uses only the plain blit/fill primitives:

- `Sprite.Draw` / `Spritesheet.Draw` — `ebiten.DrawImageOptions` (position + scale, no shader).
- `Block.Draw` / `Ball.Draw` — `vector.FillRect` / equivalent solid-color fill.
- `resource.Manager` loads fonts, images, and audio, but has no notion of loading/compiling shader source.

This came up while scoping visual effects for the Metal Slug demo (docs/game_concept_metal_slug_demo.md) — things like a hit-flash on enemy damage, a muzzle-flash glow, or a palette-swap are natural fits for a shader, but there's currently no path in the engine to load one, attach it to a draw call, or drive per-frame uniforms from Go.

## Current state

- No `.kage` shader source files anywhere in the repo.
- No `Shader` field on any component (`Sprite`, `Spritesheet`, `Block`, `Ball`).
- No shader loading/caching in `resource.Manager` (compare to how it already caches images/fonts/audio).
- No hook in `object.Drawer` or `Manager.Draw` for a full-screen post-processing pass (e.g. rendering the world to an offscreen buffer and running a shader over the whole frame — the same `worldBuffer` pattern camera-follow (PR #11) already introduces for scrolling would double as the target for a screen-space effect).

## Target state

Two distinct capabilities, likely worth separate follow-up work rather than one PR:

1. **Per-object shaders** — a `Shader *ebiten.Shader` (or a name/handle resolved via `resource.Manager`) on relevant components, with `Draw` routing through `DrawRectShader`/`DrawTrianglesShader` instead of `DrawImageOptions` when set. Needed for effects tied to one sprite (hit-flash, palette swap, outline).
2. **Full-screen post-processing** — a shader pass over the already-composited frame (or the `worldBuffer` camera-follow already renders into), for effects like vignette/CRT/color-grade that aren't tied to a single object.

Suggested shape, following the same pattern as font/audio loading:

- Add shader loading to `resource.Manager` (compile `.kage` source via `ebiten.NewShader`, cache by path, mirroring `LoadFont`/`LoadAudio`).
- Add an optional `Shader` reference to `Sprite`/`Spritesheet` (and maybe `Block`), defaulting to nil (today's plain-blit behavior) so existing scenes are unaffected.
- Prototype against one concrete effect first (e.g. hit-flash for the Metal Slug enemy, once step 5 of that plan exists) rather than building the abstraction speculatively — mirrors how the process manager (PR #14) landed as a tested primitive before any consumer adopted it, and avoids designing the uniform-passing API against a hypothetical use case.

## Effort / priority

- **Effort**: Medium
  - Ebiten's shader API itself is small; most of the work is deciding where shader references live on components and how uniforms get updated per frame (e.g. does a component own a `time float64` uniform itself, or does something external drive it).
  - No changes needed to existing draw paths until a component opts in (`Shader == nil` keeps today's behavior).
- **Priority**: P2 (medium)
  - Not blocking any currently-planned engine work, but flagged directly by the Metal Slug demo's visual-effects wishlist (hit-flash, muzzle-flash) — likely wanted well before that plan's later steps (enemy death, polish pass).
