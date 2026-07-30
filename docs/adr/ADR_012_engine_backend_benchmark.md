# ADR-012: Config-selectable engine/rendering backend for benchmarking

## Status

Proposed. Not started — this ADR is a cost/approach estimate to support a go/no-go decision, not a committed design.

## Context

The idea floated: let a field in the existing `config.yaml` (alongside `script_engine: "lua"|"python"`,
which already does exactly this kind of backend selection) pick which implementation actually runs the
game — today's Go/Ebiten stack, or a C++ alternative — so the two can be benchmarked against each other.
Two rendering options were proposed alongside today's Ebiten: raw OpenGL, and
[raylib](https://www.raylib.com/). Both have mature, pre-built Go bindings
(`gen2brain/raylib-go`, `go-gl/gl`) — see Approach 1 below — so neither requires hand-writing C++;
a from-scratch **C++ engine** (Approach 2) is a separate, much larger question about the language
and runtime, not about which rendering library is used.

Before costing this, it's worth being precise about what "swap a layer" can mean here, because the
codebase does not have a clean seam at the rendering boundary today:

- **`*ebiten.Image` is the asset type itself**, not an abstraction over one. It appears directly in
  `object.Component`'s `Drawer` interface (`Draw(screen *ebiten.Image, transform *Transform)`), in
  `Sprite`, `Spritesheet`, `Block`, `Ball`, and in `resource.Manager`'s cache. Swapping the renderer
  means touching every component that draws, not one file.
- **Ebiten owns the main loop.** `frameengine/application/engine/engine.go` calls
  `ebiten.RunGame(e.game)`, which is inversion-of-control: Ebiten's runtime calls back into
  `Game.Update/Draw/Layout`. Nothing in this engine drives its own for-loop today.
- **Physics is not a cost driver either way.** `github.com/oliverbestmann/box2d-go` (see its
  `transpile.sh` / `box2d-3.1.1.patch`) is a pure-Go *transpile* of the real upstream Box2D v3.1.1
  C++ library — not a from-scratch Go physics engine. A native C++ build would link the real Box2D
  directly, which is **less** code than what we carry today, not more. Our own glue around it
  (`frameengine/physics/*.go`, non-vendored) is 95 lines.
- **Scripting is the opposite case.** `frameengine/script/` (1,067 lines) wraps two *pure-Go*
  VMs — `gopher-lua` and `gpython` — because that's what let this engine avoid CGo. A C++ build
  would instead embed the real Lua 5.1 C API and/or CPython, which is standard and arguably easier
  to embed than to reimplement — but it is not a line-for-line port; it's a re-architecture of the
  engine-API binding glue (`play_sound`, `switch_scene`, `emit`, `self.*`, `engine.*`).
- **The WASM build is Ebiten-shaped too.** `main_wasm.go` / `games/demo1`'s `embed.FS` loading path
  ships today via Ebiten's WASM support. Neither C++ option gets that for free — it would need
  Emscripten, a separate toolchain decision not covered by this ADR.

Rough current sizes (non-test Go, excluding the vendored `box2d-go` dependency):

| Package | Lines | Rendering-backend-specific? |
|---|---:|---|
| `object/` (Component/GameObject system) | 710 | No |
| `event/` + `events/` (bus + event types) | 256 | No |
| `application/data/` (YAML scene loader) | 453 | No |
| `physics/` (our glue around box2d-go) | 95 | No |
| `script/` (Lua + Python VM hosting) | 1,067 | No (but not a straight port — see above) |
| `resource/` (asset cache: images/audio/fonts) | 307 | **Yes** |
| `view/` (scenes, camera, UI, input, physics debug draw) | 1,340 | **Yes** |
| **Total** | **5,338** | |

The rendering-specific slice (`resource/` + `view/`, ~1,650 lines) is the only part that differs
between an OpenGL build and a raylib build. Everything else (~2,585 lines of `object`/`event`/
`application/data`/`physics`/`script`, plus their tests) would need to exist, unchanged in scope,
in *both* C++ variants if the goal is a full engine rewrite.

---

## Approach 1: Swap only the render/input/audio layer, keep Go for everything else

**Idea:** Keep the entire simulation (`object/`, `event/`, `physics/`, `script/`,
`application/data/`) exactly as-is in Go. Replace only `view/` + `resource/`'s Ebiten calls with
calls into a different rendering library — reachable entirely through existing, mature Go
bindings, with **no hand-written C++ or cgo code required on our side**:

- **raylib variant:** [`gen2brain/raylib-go`](https://github.com/gen2brain/raylib-go) is a
  complete, actively maintained cgo binding to the real raylib C library — textures, 2D drawing,
  input polling, audio, and window/context management all come through Go function calls
  (`rl.DrawTexture`, `rl.PlaySound`, `rl.IsKeyDown`, ...). We would write Go code against this
  binding, the same way `view/scene` today writes Go code against `ebiten.Image`. No custom cgo,
  no separate build toolchain, still `go build`.
- **OpenGL variant:** equally, [`go-gl/gl`](https://github.com/go-gl/gl) (generated bindings to
  the real OpenGL C API) + [`go-gl/glfw`](https://github.com/go-gl/glfw) (window/context/input) are
  the Go-idiomatic path — again no C++ to write. Unlike raylib, `go-gl` only gives you the raw GL
  calls: no 2D sprite batcher, no texture decoder, no audio. We'd still use Go's stdlib
  `image`/`image/png` for texture decoding and could reuse Ebiten's own audio dependency
  ([`hajimehoshi/oto`](https://github.com/hajimehoshi/oto), a thin cgo/platform-audio binding) to
  avoid writing an audio layer from scratch — but the sprite batcher and shader pipeline are on us.

The config field (`render_backend: "ebiten" | "raylib" | "opengl"`) selects a Go-side
implementation of a new, narrow `Renderer` interface at startup; a build tag or a factory map
(same pattern as `main.go`'s `sceneFactories`) picks the concrete type.

Concretely, this means:

- Defining a `Renderer` interface (draw sprite/rect/text, load texture, play sound, poll input,
  own the window) that `resource.Manager` and the `Drawer`/UI code call through, instead of the
  concrete `*ebiten.Image` type.
- Dropping `ebiten.RunGame`'s inversion of control for the non-Ebiten backends: `application/game`
  would drive its own `for { Update(dt); Draw() }` loop when `render_backend != "ebiten"`.
- Reimplementing `resource/image.go`, `resource/audio.go`, `resource/font.go`, `view/scene`'s Draw
  paths, `view/ui`, and `view/input` against the new interface — the ~1,650 lines above, once per
  variant.

**What this benchmarks:** whether Ebiten's Go-side rendering/draw-call path is a bottleneck
compared to calling into a mature native rendering library from Go, at a fixed simulation cost
(same ECS, same physics, same GC pressure — all untouched Go). This is a fair, apples-to-apples
test of *rendering*, and only rendering.

**What this does NOT tell you:** anything about whether C++ *as a language/runtime* would be
faster overall. Both variants here stay Go programs start to finish — cgo calls into raylib or
into OpenGL, not a rewrite in C++. That question is only answered by Approach 2, below.

**Cost:** Medium-low. Days to ~1 week for the raylib variant — the binding is pre-built, tested,
and documented, so the work is almost entirely "implement our `Renderer` interface against
raylib's API," not "build a rendering library." The OpenGL variant costs more on top of that: no
binding gives us 2D drawing, input, or audio for free the way raylib does, so a sprite
batcher/shader pipeline has to be written by hand even though the *bindings* themselves require no
custom cgo. Realistically add ~1–2 weeks on top of the raylib variant's cost for OpenGL
specifically.

---

## Approach 2: A full, separate C++ engine (two variants)

**Idea:** Take the "config field picks the runtime" idea literally at the process level: the field
selects which *executable* runs (Go binary, or a from-scratch C++ engine), because a real C++
engine can't be dynamically loaded into a running Go process without an FFI layer that would
itself dominate the engineering cost. This means re-implementing, in C++:

- The Component/GameObject actor system (`object/`, 710 lines) — including Prototype-clone
  spawning, the `Timer`/`Enemy`/`Projectile` component shapes, and everything
  `application/data/builders.go` currently does to turn YAML into typed components.
- The event bus (`event/` + `events/`, 256 lines) — type-safe dispatch, deferred queue.
- The YAML scene/config loader (`application/data/`, 453 lines) — via `yaml-cpp` or similar, then a
  builder-registry pattern equivalent to today's.
- A script host embedding real Lua 5.1 and/or CPython, replacing `script/`'s two pure-Go VMs, and
  re-implementing every `engine.*`/`self.*` binding every existing Python script in
  `games/metalslug_demo/scripts/python/` depends on (`get_timer`, `set_facing`, `play_animation`,
  `apply_linear_impulse_to_center`, ...) — this is the part with the least code reuse potential.
- A physics wrapper around the *real* upstream Box2D C++ (net win vs. today's transpiled Go port).
- Window/rendering/input/audio — this is where OpenGL vs. raylib actually differs (same delta as
  Approach 1's ~1,650 lines, described above), but now built on top of a from-scratch engine rather
  than reusing Go's.

To isolate the OpenGL-vs-raylib variable specifically (the original ask), this core (~2,500+ lines
of non-rendering logic) would need to be built **twice** — once per rendering choice — unless the
C++ engine is itself designed with a renderer seam from day one (which is more upfront design work,
not less).

**What this benchmarks:** whether C++ as a whole is faster/lighter than Go for this kind of engine
— GC pauses, component-iteration overhead, script-call overhead, everything. A much bigger, more
interesting number, but a much bigger project to get there.

**What this does NOT get you for free:** WASM (would need Emscripten for either C++ variant — untested
territory here), and parity with every existing YAML scene / Python script feature (`sphere_timer.py`,
`enemy_bomber.py`'s parabola math, the `BeginContact`/`EndContact` grounding logic, etc. all assume
the current script API shape) — those would need to be re-validated against the new script host, not
just recompiled.

**Cost:** Very high. This is not a spike, it's a second engine. Even a minimal vertical slice that
reproduces `metalslug_demo`'s current feature set (component system, prototype spawning, Box2D
physics, one scripting language, YAML-driven scenes, camera-follow, spritesheet animation) is
realistically a multi-week solo effort before any benchmarking can start — likely 1–2 months for
one rendering variant, plus the 1–2 extra weeks Approach 1 quantified for OpenGL vs. raylib
specifically, if both variants are wanted.

---

## Comparison at a glance

| | Approach 1 (render-layer swap, Go stays) | Approach 2 (full C++ engine) |
|---|---|---|
| **What changes** | `view/` + `resource/` only (~1,650 lines) | Everything (~5,300+ lines), rebuilt in C++ |
| **What's reused** | 100% of ECS, physics, scripting, event bus | Only the *design* (YAML schema, component shapes) |
| **Answers** | "Is our rendering path the bottleneck?" | "Is C++ overall faster/lighter than Go here?" |
| **WASM** | Unaffected (Ebiten path untouched; new backends need not support WASM) | Needs Emscripten investigation for either variant |
| **Cost, raylib variant** | Days–1 week (pre-built Go binding, no cgo of our own) | 1–2 months (engine) + relatively small renderer delta |
| **Cost, OpenGL variant** | + 1–2 weeks on top of raylib's cost (own sprite batcher/audio wiring; bindings themselves are still off-the-shelf) | + same relative delta, on top of a much larger base |
| **Risk of wasted effort** | Low — reusable regardless of the answer | High if the eventual answer is "not worth a rewrite" |

## Recommendation

Build **Approach 1** first, as a real (not hypothetical) spike: add the `render_backend` config
field now (trivial — same pattern as `script_engine`), define the narrow `Renderer` interface, and
implement the raylib variant only initially, since it's the cheaper of the two and already proves
out the interface boundary. Benchmark frame time (p50/p95), draw-call throughput at increasing
entity counts, and GC-attributable frame spikes, using the *same* `metalslug_demo` scene/YAML on
both backends.

Only commission Approach 2 — and only decide OpenGL vs. raylib for it — if Approach 1's numbers
show the *rendering* path, specifically, is the actual bottleneck (e.g. Ebiten's draw calls scale
badly past N sprites while Update/Step stay cheap). If Approach 1 shows the simulation side
(component iteration, physics step, script dispatch) dominates frame time instead, that's evidence
a full C++ engine would need to rewrite the *simulation*, not the renderer, to pay off — a
different, larger ADR than this one, and not something either C++ rendering variant would fix.

## Open questions before starting Approach 1

- Exact shape of the `Renderer` interface (how much of Ebiten's `DrawImageOptions`-style transform
  API needs to survive vs. how much can be simplified now that we control both callers).
- Whether `application/game.Game` should keep implementing `ebiten.Game` unconditionally (with
  non-Ebiten backends driving their own loop underneath a shim), or whether the IoC inversion
  needs to move up a level in `application/engine`.
- Whether benchmark scenes should be capped to the subset of components both backends can draw
  (e.g. does the physics debug-draw path in `view/scene/physics_system.go` need a raylib
  equivalent from day one, or can it be Ebiten-only during the spike).

## References

- ADR-003: Layer separation — the render backend lives entirely inside the View layer.
- `docs/frame_engine_migration_plan.md`: the ongoing `frameengine/` extraction this benchmark
  would sit alongside (a `Renderer` interface is also a prerequisite for a clean public API if
  `frameengine` ever ships as an importable module supporting more than one renderer).
