# ADR-010: Engine metrics and observability

## Status

Proposed.

## Context

We want visibility into the engine while it runs: live GameObject counts, event bus queue depth/lag, process manager activity, memory usage, goroutine/channel activity, and metrics from the embedded scripting languages (Lua via `gopher-lua`, Python via `gpython`).

This ADR surveys the realistic options — including generic Go tooling, the broader observability/APM market, and prior art from other game engines — before picking a direction.

### Constraints that shape this decision

- **No per-frame logging** (`CLAUDE.md` / ADR-006): whatever we build must not add `log.Printf`-per-frame overhead to `Update`/`Draw`.
- **No globals** (`CLAUDE.md`): config, bus, and engine are constructed and injected, never package-level mutable state. This rules out the idiomatic use of stdlib `expvar` (a global `expvar.Map`) without a wrapper.
- **Pure-Go dependencies, deliberately.** `gopher-lua` and `gpython` were chosen specifically because they're pure Go (no cgo), which is what makes WASM builds (`main_wasm.go`, `build_wasm.sh`) possible at all. Any observability tool that requires cgo bindings is a non-starter for the same reason.
- **Single local process, not a fleet.** This is a game running on one machine (or in a browser tab via WASM), not a server farm. Tooling built for monitoring many long-running service instances brings cost and complexity this project doesn't have a use for.
- **What's already free:** `object.Manager.Objects()` (→ `len(...)` for a GameObject count), a `process.Manager.Count()`, and Ebiten's own `ActualFPS()`/`ActualTPS()`. None of these currently feed anywhere visible.
- **What's genuinely missing:** the event bus (`event/bus.go`) exposes no queue-depth introspection; the script engines (`script/lua_engine.go`, `script/python_engine.go`) have zero instrumentation (no call counts, no timings); and — worth being honest about — the engine has almost no goroutines/channels to begin with (just the SIGINT/SIGTERM shutdown pair in `application/engine`), so "channel monitoring" has little to actually observe today.
- **A word on "queue lag/retries":** the event bus is fully synchronous and single-threaded ("No goroutines or channels are used", per its own doc comment) — events emitted during handler execution are queued and drained before `Emit` returns. There is no persistent cross-frame backlog and no retry concept to monitor in the way an async message queue would have. The meaningful signal here is **peak re-entrant queue depth per `Emit` call** (how deep did handler-triggered-handler recursion get), not "lag" in the networked-queue sense.

## Approach 1: In-engine debug overlay only

Extend the existing `DebugOverlayToggled` toggle (currently just a physics-shapes overlay, bound to F3) into a real HUD that renders live numbers as text every frame it's active.

| Pros | Cons |
|------|------|
| Zero dependencies; already wired to an input binding and a scene. | No history/trends — only "right now". |
| Respects "no per-frame logging" (draws text, doesn't log). | Resets on every restart; nothing persisted for later analysis. |
| Works identically in WASM builds (no server needed). | Manual layout work for every new metric shown. |

## Approach 2: stdlib `expvar` + `net/http/pprof`

`expvar` publishes named counters/gauges as JSON over HTTP; `pprof` gives CPU/heap/goroutine profiling, both stdlib, zero third-party dependencies.

| Pros | Cons |
|------|------|
| Zero dependencies, well-understood, part of the standard library. | `expvar`'s idiomatic usage is a package-level global registry — conflicts with this repo's own "No Globals" rule unless wrapped. |
| `pprof` is excellent for the profiling half of this problem (allocation pressure, goroutine leaks) regardless of what else we pick. | Needs an HTTP listener, which doesn't exist (and doesn't make sense) in a WASM build — would need the same js/notjs build-tag split already used for `application/engine/signals_*.go`. |
| Can run only in debug builds, at negligible cost. | JSON snapshot only — no graphing/history without external tooling polling it yourself. |

## Approach 3: Market APM / observability stacks (Prometheus, OpenTelemetry, Datadog/New Relic, Sentry Performance)

The general-purpose "market" options for production service observability.

| Pros | Cons |
|------|------|
| Mature, well-documented, industry-standard instrumentation APIs (counters/gauges/histograms, spans). | Built for fleets of long-running services with a scraper/collector infrastructure (Prometheus+Grafana, or a vendor backend) — there's no fleet here, just one local process or a browser tab. |
| Prometheus's `client_golang` and OpenTelemetry's Go SDK are both solid if we ever run a server component (e.g. a future authoritative-server multiplayer mode, ADR-004). | Heaviest dependency footprint of any option here; OpenTelemetry especially pulls in a large SDK surface for distributed tracing we don't do. |
| Sentry has a Go SDK with performance-transaction tracing, useful for crash/error telemetry from *shipped* games. | Commercial backends (Datadog, New Relic, Sentry's hosted tier) mean cost and a network dependency — the opposite of what a local dev/debug tool needs. Also solves a different problem: player-facing crash/analytics telemetry, not engine-internals introspection during development. |

## Approach 4: Game-engine-specific prior art (Tracy Profiler, Godot's `Performance` singleton, Bevy's diagnostics plugin)

Looking at what other engines actually ship, rather than generic APM:

- **[Tracy Profiler](https://github.com/wolfpld/tracy)** — a widely-used real-time frame profiler in the C/C++ game industry (frame timings, memory, lock contention, GPU zones), with a live desktop viewer. Go bindings exist but require cgo to talk to Tracy's C capture protocol — directly conflicting with this project's pure-Go/WASM constraint. Good inspiration for *what a frame-level profiler view looks like*, not adoptable as-is.
- **Godot's built-in `Performance` singleton** — ships in-engine "monitors" (FPS, draw calls, physics, memory, object counts) viewable in the editor's Monitor panel, plus `Performance.add_custom_monitor(name, callable)` for game code to register its own named values. This is architecturally very close to what we're describing: a small in-process registry of named values, no external service, feeding a built-in viewer.
- **Bevy's `bevy_diagnostic` crate** — a generic `DiagnosticsStore` that any subsystem can push named `f64` measurements (with a short rolling history) into, consumed by a `LogDiagnosticsPlugin` or an overlay. Same shape again: an internal registry as the primitive, with pluggable consumers (log line vs. on-screen overlay).

| Pros | Cons |
|------|------|
| Godot's and Bevy's designs validate the "small internal registry + pluggable viewer" shape directly, from engines solving the exact same problem. | None of these are Go libraries we can `go get` — they're designs to learn from, not dependencies to adopt. Tracy specifically is cgo-only, ruling it out concretely. |
| Confirms this is a well-trodden pattern, not a novel one we'd be inventing from scratch. | Still requires us to build the Go-side registry ourselves either way. |

## Approach 5: Custom lightweight metrics registry (recommended)

An explicitly-constructed (not global) registry of named counters/gauges — matching this project's own precedent of preferring small custom types over heavy dependencies (custom `vec2` over `go-gl/mathgl`; the `physics/box2d` anti-corruption wrapper). Fed by:

- `object.Manager` and `process.Manager` — already expose what's needed (`Objects()`, `Count()`); just wire the numbers through.
- A small addition to `event.Bus` — track peak re-entrant queue depth per `Emit`, exposed via a method, not a global.
- A thin instrumented wrapper implementing `script.Engine` — wraps a real `LuaEngine`/`PythonEngine` and records call counts/timings for `DoFile`/`DoString`/`CallScriptUpdate`/`CallOnEvent`, added only when instrumentation is enabled (decorator pattern, no changes needed inside the real engines).
- `runtime.NumGoroutine()` / `runtime.ReadMemStats()` sampled at low frequency (e.g. once/second via a ticked process — a natural first real consumer for the process manager, PR #14) rather than every frame.

Consumed initially by the Approach-1 debug overlay; a `pprof`/`expvar`-style snapshot endpoint (Approach 2) could be added later as an optional, non-WASM-only extra without changing the registry itself.

| Pros | Cons |
|------|------|
| No dependencies; matches every existing project convention (no globals, small interfaces, pure Go, works in WASM). | We own all of it — no community maintenance, no existing bug fixes to inherit. |
| Exactly the shape validated by Godot/Bevy's prior art (Approach 4), without adopting a non-Go dependency. | More upfront design work than "just add a library" — need to decide the registry's API before any consumer can use it. |
| Feeds the existing debug overlay with no new infrastructure (no HTTP server, no external service). | Doesn't get us distributed tracing or long-term historical storage "for free" the way a market APM stack would, if that's ever actually needed later. |

## Comparison at a glance

| | Overlay only | expvar+pprof | Market APM | Engine prior art (Tracy/Godot/Bevy) | Custom registry |
|--|--|--|--|--|--|
| **Dependencies** | None | Stdlib only | Heavy (client libs + backend) | Tracy: cgo (ruled out); Godot/Bevy: none (not Go libs) | None |
| **Fits "No Globals"** | Yes | Only if wrapped | Yes (client libs are injectable) | N/A | Yes, by design |
| **Fits WASM build** | Yes | Partial (pprof/HTTP needs build-tag split) | No (needs network egress to a backend) | N/A | Yes |
| **History / trends** | No | JSON snapshot only | Yes (that's the point) | Varies | No (unless we add it later) |
| **Fits "single local process"** | Yes | Yes | No — built for fleets | N/A | Yes |
| **Effort** | Low | Low–Medium | Medium–High (infra to run/pay for) | N/A (reference only) | Medium |

## Recommendation

Build the **custom lightweight registry (Approach 5)**, feeding the **existing debug overlay (Approach 1)** as the first consumer. Add **`net/http/pprof`** behind a debug build flag for ad hoc profiling (Approach 2's profiling half) — it's free, stdlib, and solves a genuinely different problem (allocation/goroutine profiling) than live counters do. Explicitly do **not** adopt Prometheus, OpenTelemetry, Datadog/New Relic, Sentry, or Tracy for engine-internals monitoring at this stage (Approaches 3–4) — they solve problems (fleet monitoring, distributed tracing, cgo-based frame capture) this project doesn't have, and several conflict directly with the pure-Go/WASM constraint. Revisit market APM options only if/when a server component actually exists (e.g. an authoritative-server multiplayer mode, ADR-004) and there's a real fleet to observe.

## References

- ADR-006: Coding standards (no per-frame logging, no globals, minimize allocations in Update/Draw).
- ADR-002 / ADR-003: Event bus design and layer separation (why the bus has no goroutines/channels today).
- ADR-007: Object pool pattern (same "measure before adding infrastructure" philosophy).
- `vec2` package PR: precedent for choosing a small custom type over a heavier third-party dependency.
- Process manager (`process` package) PR: precedent for landing a tested, standalone primitive before any consumer adopts it — the same shape this ADR proposes for the metrics registry.
