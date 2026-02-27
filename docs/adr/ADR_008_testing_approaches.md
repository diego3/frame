# ADR-008: Testing approaches for modules and components

## Status

Proposed.

## Context

We need a clear strategy for testing the engine and game: which kinds of tests to write, when to use them, and what tradeoffs they bring. Different approaches suit different layers (Application, Game Logic, Game View) and different goals (regression safety, determinism, performance, correctness).

This ADR describes testing approaches that fit our architecture (ADR-003: layers and events; ADR-006: coding standards). It includes standard practices (unit, integration) and options common in game development (determinism, property-based, visual regression, playtesting). The team can adopt a subset and expand over time.

---

## Approach 1: Unit tests

**Idea:** Test a single unit (function, type, or small package) in isolation. Dependencies (event bus, physics backend, config) are replaced with mocks or fakes. No real game loop, no rendering, no I/O.

**Good for:** Game Logic rules, event handlers, pure functions (e.g. damage formula, score calculation), and any code that can be exercised by calling functions and asserting return values or state.


| Pros                                                                                                 | Cons                                                                     |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| Fast feedback; run in milliseconds.                                                                  | Does not catch bugs at boundaries between real components.               |
| Easy to pinpoint failures; one unit under test.                                                      | Mocks can drift from real behavior (mock does not match real event bus). |
| Works well with event-driven Logic: emit intents, assert state or emitted events (ADR-003, ADR-006). | Some code is hard to unit test (e.g. tight coupling to Ebiten or GPU).   |
| Fits table-driven tests and subtests (`t.Run("scenario", …)`).                                       | Writing mocks and fakes adds code to maintain.                           |


**Technology options**


| Tech               | Summary                                                                                                          |
| ------------------ | ---------------------------------------------------------------------------------------------------------------- |
| `testing` (stdlib) | Go’s built-in test runner; `go test`, table-driven tests, subtests.                                              |
| `testify`          | Assertions (`assert`, `require`) and test suites; reduces boilerplate. Used for most unit and integration tests. |
| `gomock`           | Generate mocks from interfaces; use for event bus, ports, and other dependencies where behavior matters.         |
| `go-mock` (uber)   | Hand-written mocks with a small API; no codegen. Optional, use only if it fits better than gomock.               |


**Summary:** Use for the majority of regression coverage. Prefer testing Logic via events (emit intents, assert state) rather than full engine bootstrap.

---

## Approach 2: Integration tests

**Idea:** Test several packages together with real (or near-real) dependencies. For example: real event bus, real scene manager, real physics adapter—but optionally no window, or headless/minimal renderer. The game loop may run for a limited time or number of frames.

**Good for:** Cross-layer flows (e.g. intent event → Logic → state event → View subscription), resource loading, scene transitions, and any behavior that only appears when components are wired as in production.


| Pros                                                                                  | Cons                                                                |
| ------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Catches bugs that unit tests miss (wrong wiring, wrong event types).                  | Slower than unit tests; may need setup (temp files, test assets).   |
| Builds confidence that the system works end-to-end for a slice of behavior.in         | Failures can be harder to diagnose (which component is wrong?).     |
| Can reuse production wiring with minimal overrides (e.g. no window, test asset path). | Risk of flakiness if tests depend on timing or unmanaged resources. |


**Summary:** Use for critical paths that span layers (e.g. “scene change requested → scene actually changes”) and for adapter integration (e.g. physics, resources). Keep the number of integration tests manageable; prefer few, focused scenarios.

---

## Approach 3: Determinism and snapshot tests

**Idea:** Run the same inputs (fixed seed, fixed delta time, recorded intents) and assert that the outcome is identical (e.g. same state hash or same key values after N frames). Useful for replay, network sync (ADR-004), and regression: “this run must match the golden snapshot.”

**Good for:** Game Logic that must be deterministic (no `rand` in logic, or seeded RNG; no map iteration where order matters). Validates that a change did not alter behavior for a given input sequence.


| Pros                                                                                             | Cons                                                                      |
| ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------- |
| Catches subtle non-determinism (e.g. map iteration, unseeded RNG) that can break replay or sync. | Golden snapshots can be large or brittle; format changes require updates. |
| Single source of truth: “this input sequence produces this state.”                               | Only as good as the input sequences you record; may miss untested paths.  |
| Aligns with ADR-006 (determinism where replay/network may be required).                          | Can be slow if the run is long (many frames).                             |


**Technology options**


| Tech           | Summary                                                                                                 |
| -------------- | ------------------------------------------------------------------------------------------------------- |
| `go-snaps`     | Jest-style snapshot testing; save and diff text/JSON/YAML output; good for state or event stream dumps. |
| `go-cmp`       | Flexible equality and diffs for structs; use for custom state comparison and clear failure messages.    |
| Custom hashing | Hash state (e.g. `crypto/sha256` of serialized state) and compare to golden hash; minimal dependencies. |


**Summary:** Use where determinism is required (replay, networking) or where you want to lock behavior for a critical scenario. Start with a small set of short, deterministic runs and compare state (or event stream) to a stored snapshot.

---

## Test file location: in-package vs centralized tests folder

We need a consistent convention for where Go tests live in this project.

### Option A: In-package tests (`*_test.go` next to code)

Tests live in the same package directory as the code (e.g. `event/bus.go` and `event/bus_test.go`).

**Pros**

- **Local feedback:** Tests sit next to the code they exercise; easier to discover and maintain.
- **Shared imports and helpers:** Tests can share test helpers and fixtures per package without extra plumbing.
- **Standard Go workflow:** Works naturally with `go test ./...`; aligns with Go tooling and documentation.
- **Package-private coverage (optional):** Using the same package name in tests (`package event`) allows testing unexported behavior when justified.

**Cons**

- **Mixed concerns in one folder:** Source and tests are interleaved in the same directory (though Go tools handle this well).
- **Risk of over-testing internals:** Same-package tests can encourage reaching into unexported details instead of testing behavior.

### Option B: Centralized `tests/` folder

Tests live in a separate top-level directory (e.g. `tests/`), importing the public API of packages under test.

**Pros**

- **Clear separation:** Source tree under `goengine/` contains only production code; `tests/` contains only tests.
- **Public-API focus:** Tests must go through exported functions/types, encouraging better encapsulation.
- **Cross-package scenarios:** Easier to write “black-box” tests that exercise multiple packages together, similar to integration tests.

**Cons**

- **More boilerplate:** Tests must import packages and set up wiring that in-package tests would get “for free.”
- **Weaker locality:** Harder to find tests for a specific type or function; small units can get lost in a large tests folder.
- **Non-idiomatic for small/medium Go modules:** Most Go projects keep tests next to the code; contributors may expect that layout.

### Decision

- **Default:** Keep tests in the same package directory as the code (`*_test.go` next to `*.go`), using `testing` + `testify` as the primary tools.
- **Exceptions:** Use a separate `tests/` directory only for rare, cross-package integration or system-level tests that do not fit cleanly inside one package. Even then, prefer reusing production wiring rather than duplicating it.

---

## Approach 4: Property-based tests

**Idea:** Instead of hand-written examples, define *properties* (invariants) that should hold for many inputs. A test runner generates random (or exhaustive) inputs and checks the property. Example: “physics velocity magnitude is never NaN or infinite,” or “score never decreases after a positive event.”

**Good for:** Math, physics, formulas, and any code where invariants are easier to state than a full set of examples. Complements unit tests by stressing edge cases you might not think of.


| Pros                                                                   | Cons                                                                              |
| ---------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Finds edge cases (zeros, negatives, very large numbers) automatically. | Requires thinking in terms of invariants; not all code has simple properties.     |
| One property can replace many example-based tests.                     | Failures may show a minimal input that is hard to interpret; need good shrinking. |
| Works well with numeric and deterministic logic.                       | Less natural for highly stateful or event-sequence-dependent behavior.            |


**Technology options**


| Tech                         | Summary                                                                                      |
| ---------------------------- | -------------------------------------------------------------------------------------------- |
| `rapid` (pgregory.net/rapid) | Property-based testing with shrinking; type-safe generators and state-machine testing.       |
| `gopter`                     | QuickCheck-style; generators and properties; more control, slightly more verbose than rapid. |
| `testing/quick` (stdlib)     | Minimal property checks; no shrinking; good for simple "holds for random inputs" checks.     |


**Summary:** Use for physics, damage/score formulas, and utilities with clear invariants. Use a Go library (e.g. `github.com/leanovate/gopter` or similar) and keep properties focused and fast.

---

## Approach 5: Visual / screenshot regression tests

**Idea:** Run the game (or a scene) for a fixed number of frames with fixed input, capture a screenshot (or frame buffer), and compare it to a stored reference image. Fail if the diff exceeds a threshold.

**Good for:** UI layout, rendering pipeline, and “does the screen look correct?” regression. Optional in many projects; more common in UI-heavy or art-heavy games.


| Pros                                                                      | Cons                                                                       |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| Catches visual regressions (wrong layout, missing sprite, broken shader). | Fragile: resolution, font rendering, GPU drivers can cause false failures. |
| Documents intended look for a scenario.                                   | Slow and resource-heavy (need display/GPU or software renderer).           |
| Can be run in CI with a headless or offscreen renderer.                   | Thresholds and diff tools need tuning; maintenance cost can be high.       |


**Technology options**


| Tech             | Summary                                                                                               |
| ---------------- | ----------------------------------------------------------------------------------------------------- |
| `image` (stdlib) | Decode reference PNG, compare pixel-by-pixel or with a tolerance; Ebiten can render to `*image.RGBA`. |
| Custom diff      | Hash or compare subregions; allow per-pixel or per-channel tolerance to reduce flakiness.             |
| External tools   | ImageMagick, perceptual diff tools; run from tests via `exec` if CI supports it; heavier setup.       |


**Summary:** Consider only if visual correctness is critical and you have CI that can run headless rendering. Prefer lower-cost tests (unit, integration, determinism) first.

---

## Approach 6: Simulation and stress tests

**Idea:** Run the game or a subsystem for a long time (many frames, many entities, or many events) without asserting specific values. Goal: no panics, no runaway memory, no deadlocks. Optionally assert invariants (e.g. “entity count stays bounded”) or collect metrics (allocations per frame).

**Good for:** Object pools (ADR-007), event bus under load, scene lifecycle, and finding leaks or instability that only appear over time.


| Pros                                                      | Cons                                                                         |
| --------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Finds leaks, goroutine leaks, and gradual degradation.    | Does not verify correctness; only that the system does not crash or explode. |
| Can run in background or overnight.                       | May need time limits and careful cleanup to avoid hanging CI.                |
| Complements correctness tests with “does it stay stable?” | Failures may be non-deterministic (timing, GC).                              |


**Technology options**


| Tech               | Summary                                                                        |
| ------------------ | ------------------------------------------------------------------------------ |
| `go test -count=N` | Run the same test N times to surface flakiness or rare panics.                 |
| `go test -race`    | Data-race detector; run periodically or in CI to catch concurrent access bugs. |
| `go test -timeout` | Cap test duration so stress tests do not hang CI.                              |
| `runtime/pprof`    | Profile CPU or heap during long runs; detect leaks or hot paths.               |


**Summary:** Use for stability and performance regression (e.g. “after 60k frames, memory and frame time stay bounded”). Run as a separate suite or on a schedule rather than on every commit if slow.

---

## Approach 7: Playtesting and manual testing

**Idea:** Humans play the game (or a build) and report bugs, feel, and balance. Not automated; part of the release process and feature validation.

**Good for:** Feel, balance, UX, and issues that are hard to specify in code (e.g. “jump feels floaty,” “this level is too hard”).


| Pros                                                                   | Cons                                                           |
| ---------------------------------------------------------------------- | -------------------------------------------------------------- |
| Catches design and UX issues that automated tests cannot.              | Not repeatable; not in CI; depends on availability of testers. |
| Essential for game quality beyond “does it run?”                       | Does not replace automated regression for logic and stability. |
| Complements automated tests; both are part of a full testing strategy. | No direct tradeoff with other approaches—use in addition.      |


**Technology options**


| Tech            | Summary                                                                              |
| --------------- | ------------------------------------------------------------------------------------ |
| Build artifacts | Export a test build (e.g. `go build -o playtest.exe`) for testers; no extra tooling. |
| Screen recorder | OBS, etc.; record sessions to reproduce “feel” or visual bugs.                       |
| Issue tracker   | Use labels (e.g. “playtest”) and checklists so feedback is consistent and traceable. |


**Summary:** Treat as a separate pillar: automated tests for correctness and regression; playtesting for feel, balance, and design. Document what to test (e.g. test plan or checklist) so coverage is consistent.

---

## Comparison at a glance


| Approach             | Speed    | Scope         | Best for                        | CI-friendly |
| -------------------- | -------- | ------------- | ------------------------------- | ----------- |
| Unit                 | Fast     | Single unit   | Logic, formulas, event handlers | Yes         |
| Integration          | Medium   | Multiple pkgs | Cross-layer flows, adapters     | Yes         |
| Determinism/snapshot | Medium   | Full run      | Replay, sync, behavior lock     | Yes         |
| Property-based       | Fast–med | Many inputs   | Invariants, math, physics       | Yes         |
| Visual regression    | Slow     | Rendering     | UI/art correctness              | Optional    |
| Simulation/stress    | Slow     | Long run      | Stability, leaks, performance   | Optional    |
| Playtesting          | N/A      | Full game     | Feel, balance, UX               | No          |


---

## Recommendation

1. **Default:** Rely on **unit tests** for Game Logic and critical paths; use **integration tests** for a small set of cross-layer flows and adapters (ADR-006).
2. **When adding replay or networking (ADR-004):** Add **determinism/snapshot tests** for the Logic and input sequences that must stay in sync.
3. **Where useful:** Add **property-based tests** for physics and formulas with clear invariants.
4. **Optional:** Add **simulation/stress tests** for pools and long-running stability; consider **visual regression** only if you have headless rendering and need to protect visual output.
5. **Always:** Use **playtesting** for feel and design; keep it separate from automated coverage.

Start with unit and integration tests; introduce determinism and property-based tests as the codebase and features grow. Avoid investing in visual or stress tests until the need is clear.

## References

- ADR-002: Event manager (Bus, Emit/Subscribe).
- ADR-003: Layer separation; Game Logic testable via intent events.
- ADR-004: Networking (determinism and sync).
- ADR-006: Coding standards (unit tests, table-driven tests, testing Logic via events).
- ADR-007: Object pool pattern (candidate for stress tests).

