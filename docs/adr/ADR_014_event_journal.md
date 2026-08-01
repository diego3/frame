# ADR-014: Event journal (recording and replay)

## Status

Proposed.

## Context

Comparing this engine's architecture to a classic id Software (Quake III Arena-style)
engine diagram — `Input (keyboard/mouse/UDP) -> Event Queue -> {Server "game" VM, Client
"cgame"+"ui" VM}`, with an **Event Journal** sitting off the queue in the shared "Common"
layer — surfaced a capability this engine doesn't have at all: recording what happened during a
play session so it can be replayed later. In that reference architecture, the journal exists to
support demo recording/playback and is a natural building block for deterministic-lockstep
networking.

`frame` already has the architectural piece that made comparison possible in the first place: a
single, synchronous event bus (`frameengine/event/bus.go`, ADR-002) that every intent event (`MoveRequested`,
input-driven `ScriptEmitted`, `DebugOverlayToggled`, ...) and every state event
(`GameObjectCreated/Destroyed`, `ComponentAdded/Removed`, `ScriptEmitted` from gameplay code,
physics contacts, ...) already passes through one choke point: `Bus.deliver(ev)`. That function
already does one relevant thing today — it increments a delivered-event counter consumed by the
debug overlay (`TakeEventCount`, per ADR-010's Approach 1). Recording is the same shape of idea,
one step further: instead of (or in addition to) counting what passed through, keep it.

**This ADR is speculative, not need-driven.** Nothing in `games/metalslug_demo/ideas.md` is
blocked on this; it exists because the architecture comparison exposed the gap, the same way
ADR-010 opened from "we want visibility" rather than a blocking bug. The goal here is to record
the decision — what shape, what it costs, what it would unlock — so that if a concrete trigger
shows up later (a bug that only reproduces sometimes, wanting a highlight-clip feature, or
actually starting ADR-004's peer-to-peer networking approach), the choice isn't made from
scratch under time pressure.

### Constraints that shape this decision

- **No per-frame logging/allocation overhead in `Update`/`Draw`** (ADR-006). Whatever this is,
  recording an event must be cheap — append to a growable in-memory buffer or a buffered writer,
  never a synchronous per-event disk flush inside the hot path.
- **No globals** (CLAUDE.md). A journal has to be an optional, explicitly-injected collaborator
  (e.g. `Bus.SetJournal(...)` or a wrapper), not package-level mutable state — same constraint
  ADR-010 already ran into with `expvar`.
- **Pure-Go, WASM-compatible, deliberately** (same reasoning as ADR-010: `gopher-lua`/`gpython`
  were chosen cgo-free specifically so WASM builds work at all). A WASM build has no real OS
  filesystem — writing a recording to disk is a native-build-only capability unless a browser-side
  sink (download, IndexedDB) is added separately; that integration is out of scope here.
- **Single local process today.** ADR-004 (networking) is still "Proposed," not built. A wire
  format for streaming the journal to a remote peer isn't required by this ADR, but Approach 1
  below is deliberately shaped so it wouldn't need to be redesigned if that day comes.
- **`dt` is already fixed**, not measured wall-clock time: `frameengine/application/game/game.go`'s
  `Update()` uses `const defaultDt = 1.0 / 60.0` for every scene update, not an elapsed-time
  read. That's a necessary — not sufficient — precondition for deterministic replay.
- **Determinism is not currently verified anywhere in this codebase**, and there's a concrete,
  already-present counterexample: `frameengine/view/camera/shake.go` calls `rand.Float64()` from Go's
  default, unseeded global source. Two "identical" replays driven by the same recorded inputs
  would render different camera shake. It's cosmetic (View-only; doesn't feed back into Logic
  state today), so it doesn't currently break anything — but it's live proof that nothing here
  guarantees reproducible randomness, and any future gameplay-affecting RNG (damage rolls, drop
  chance) would hit the identical problem the moment it's added.

## Approach 1: Record only intent/input events, replay by re-simulating

Hook the bus's single choke point (`Bus.deliver`, or a thin recording wrapper around `Emit`) to
append only the events that originate from *outside* the simulation — input-derived intents like
`MoveRequested` and input-bound `ScriptEmitted` — tagged with the frame index they occurred on.
Replay = fresh `Setup()`, then feed the same intents back onto the bus in order, driven by the
same fixed-`dt` loop, and let Logic re-derive everything else (physics, script state, spawns)
exactly as it did originally. This is the actual mechanism behind Quake's small `.dm_*` demo
files.

| Pros | Cons |
|------|------|
| Recordings are tiny — only player input, not derived state (this is why real Quake demos are a few KB for minutes of play). | Only correct if the simulation is bit-for-bit deterministic from those inputs alone — not true today (the unseeded-RNG example above), and cross-platform/cross-build float determinism for Box2D is a known-hard class of problem in general, not something this ADR can wave away. |
| Doubles as the actual foundation for ADR-004's peer-to-peer/input-sync networking approach — "record today, transmit tomorrow" is the same mechanism. | A single non-deterministic frame silently invalidates every subsequent frame of the replay — errors compound rather than staying locally visible, so trusting this requires an explicit self-check (e.g. hash simulation state after N frames of replay, compare against the original run) that doesn't exist yet. |
| Free byproduct: a recorded golden-path input log becomes a headless regression/smoke-test harness (record once, replay in CI, assert final state matches) — relevant to CLAUDE.md's "test the golden path" rule. | Requires an upfront determinism audit (starting with seeding `frameengine/view/camera/shake.go`'s RNG) before it can be trusted for anything beyond "probably close enough for a highlight reel." |

## Approach 2: Record every bus event, replay by direct playback (no re-simulation)

The recorder subscribes at the same `Bus.deliver` choke point, but to everything — intents *and*
state events (`GameObjectCreated/Destroyed`, `ComponentAdded/Removed`, gameplay `ScriptEmitted`,
physics contacts, ...) — writing each with its frame index. Replay doesn't re-run Logic at all;
it feeds the recorded state events straight to whatever's watching (View, a debug console) in
original order, playing back the transitions that already happened instead of recomputing them.

| Pros | Cons |
|------|------|
| Doesn't need determinism to be correct — you're replaying the literal thing that occurred, so unseeded RNG or float drift can't invalidate it. | Recordings are much bigger — every state event, not just player input; none of Approach 1's "few KB" win. |
| Simplest to implement correctly *right now*, since it needs no audit of the simulation's determinism first. | Gives ADR-004's peer-to-peer approach nothing — rollback/lockstep netcode specifically needs re-simulation from inputs, and playing back already-computed state is the wrong shape for that. |
| Immediately useful for bug reports ("here's the exact event sequence that led to the freeze") with none of Approach 1's correctness risk. | A replay is inherently watch-only — there's no way to fork it into a live session or feed different input partway through, the way an input-log replay naturally allows. |

## Approach 3: Periodic full-state snapshots (savestate-style)

Instead of (or layered on top of) an event log, periodically serialize the entire simulation
state — every `GameObject`/component plus the Box2D world — into a snapshot. A "journal" becomes
a sequence of snapshots, optionally with fine-grained events between them.

| Pros | Cons |
|------|------|
| The only approach that enables scrubbing/seeking (jump straight to frame 3000 without replaying 3000 frames first) — a genuinely different capability, closer to a rewind-debugger than a demo recorder. | By far the heaviest: needs exhaustive, kept-in-sync serialization of every `object.Component` type as new ones are added. |
| | `frameengine/physics/box2d`'s wrapper currently has no save/restore of Box2D body state at all — this approach requires building that first, which is its own significant chunk of work, not something layered on the existing `Bus.deliver` choke point like Approaches 1-2. |

## Approach 4: Don't build this now

Defer. No open `ideas.md` TODO needs this, and it surfaced from an architecture comparison, not
a blocked feature. Same discipline `process.Manager` followed (Game Coding Complete Ch. 4's
pattern landed standalone and sat with zero consumers until camera shake actually needed it) —
except this ADR chooses to record the analysis now, so a future concrete trigger picks an
already-considered approach instead of re-litigating from zero.

| Pros | Cons |
|------|------|
| Zero cost now; avoids speculative infrastructure, matching ADR-007's "measure before adding" precedent (explicitly cited by ADR-010 too). | If a hard-to-reproduce bug shows up next month, there's no tool to capture it — the cost doesn't disappear, it just moves to the least convenient possible time. |
| Doesn't force a commitment to Approach 1 before the determinism audit it needs has even been scoped. | |

## Comparison at a glance

| | Approach 1 (input + re-sim) | Approach 2 (full event log) | Approach 3 (snapshots) | Approach 4 (defer) |
|--|--|--|--|--|
| **Recording size** | Smallest (input only) | Larger (all state events) | Largest (full state per snapshot) | N/A |
| **Needs determinism** | Yes — not currently verified | No | No | N/A |
| **Implementation cost now** | Medium (recorder) + audit work (open-ended) | Low (recorder only) | High (needs Box2D save/restore first) | None |
| **Enables ADR-004 peer-to-peer netcode** | Yes — same mechanism | No | No | No (until revisited) |
| **Enables scrub/seek replay** | No (linear replay only) | No (linear replay only) | Yes | N/A |
| **Fits WASM today** | Native-build-only (no OS FS sink) | Native-build-only | Native-build-only | N/A |
| **Correctness risk** | High until audited | Low | Low | N/A |

## Recommendation

**Approach 4 (defer) for now.** There is no concrete trigger today, and Approach 1 — the option
that would actually be worth building deliberately, since it doubles as ADR-004's networking
foundation — depends on a determinism audit that hasn't even been scoped yet (starting with
seeding `frameengine/view/camera/shake.go`'s RNG and auditing for any other non-deterministic reads).
Building Approach 2 or 3 speculatively, without a real consumer, repeats the exact mistake
ADR-007's precedent warns against.

**If a concrete trigger shows up before that audit happens** (a bug report that needs exact
reproduction, wanting a highlight-clip capture feature), reach for **Approach 2** as the
pragmatic first cut — it's correct immediately, with no dependency on unverified determinism, and
the `Bus.deliver` choke point it needs already exists.

**Revisit Approach 1 specifically** if/when ADR-004's peer-to-peer/input-sync networking approach
is actually pursued — at that point the determinism audit stops being optional homework and
becomes a hard requirement of the networking work itself, so the cost is paid once for both.

## Open Questions

- **Hook point**: `Bus.SetJournal(...)` living on `Bus` itself (natural, since `deliver` already
  tracks `eventCount` the same way) vs. an external wrapper around every `Emit` call site
  (more invasive, no obvious advantage). Leans toward the former.
- **Always-on ring buffer vs. explicit start/stop**: a bounded in-memory ring buffer ("keep the
  last N seconds, dump it on crash/panic") and an explicit start/stop capture (F-key or config
  flag, written out for later sharing) solve different problems and aren't mutually exclusive —
  worth deciding both are wanted, not picking one over the other, whenever this is built.
- **Serialization format** for events carrying `map[string]interface{}` payloads (`ScriptEmitted`
  in particular): `encoding/gob` (smaller, faster, pure Go) vs. JSON (human-diffable, easier to
  eyeball while debugging a captured recording). No strong pull either way yet.
- **WASM support**: is a browser-side sink (triggering a download, or writing to IndexedDB) part
  of v1, or — like ADR-010's `pprof` — a native-build-only debug tool at first? Leans toward the
  latter, deferring WASM support until there's a WASM-specific reason to need it.

## Consequences

### Positive (once actually built, per the Recommendation above)

- A recording mechanism, once it exists, is reusable for at least three otherwise-unrelated
  needs (bug repro, highlight capture, netcode foundation) rather than being single-purpose.
- Recording via the existing `Bus.deliver` choke point requires no changes to any of the many
  `Emit`/`Subscribe` call sites across the codebase — the same "small seam, no call-site churn"
  property ADR-010's registry design leans on.

### Negative

- Deferring (this ADR's actual recommendation) means the capability doesn't exist yet — a
  hard-to-reproduce bug encountered before this is revisited still has to be chased the hard way.
- Approach 1, the most valuable long-term option, is gated on a determinism audit whose scope
  isn't yet known — it could be small (seed one RNG call) or reveal more non-determinism once
  actually looked for.

## References

- ADR-002: Event manager for decoupling subsystems (the `Bus`/`Emit`/`Subscribe` design this
  builds on).
- ADR-004: Networking approaches (peer-to-peer/input-sync explicitly needs recorded/transmitted
  input — this ADR's Approach 1 is that same mechanism, built early).
- ADR-006: Coding standards (no per-frame logging, no globals — the constraints this design has
  to fit inside).
- ADR-007: Object pool pattern ("measure before adding infrastructure" precedent, cited again
  here the same way ADR-010 cited it).
- ADR-010: Engine metrics and observability — sibling "Proposed," options-and-tradeoffs ADR;
  `Bus.deliver`'s existing `eventCount` counter is exactly the kind of small addition to the same
  choke point this ADR proposes going further with.
- `frameengine/event/bus.go`: `Bus.deliver`, the single choke point every approach above hooks into.
- `frameengine/view/camera/shake.go`: the concrete unseeded-RNG example cited under Constraints.
