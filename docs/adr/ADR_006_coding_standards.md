# ADR-005: Coding standards for the codebase

## Status

Proposed.

## Context

As the team and codebase grow, we need shared coding standards so that:

- Code is consistent, readable, and maintainable across packages.
- Go idioms and ecosystem norms are followed.
- Game-engine constraints (performance, frame budget, determinism) are respected without over-engineering early.

This ADR establishes standards the team agrees to follow. It draws from Go best practices (Effective Go, Go Code Review Comments, Standard Project Layout) and common game-development practices, adapted to our Ebiten-based engine.

## Decision

### 1. General Go style


| Area              | Standard                                                                                                                                                                                                                    | Rationale                                        |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| **Formatting**    | Use `gofmt` (or `go fmt`). No custom style.                                                                                                                                                                                 | Single canonical style; tools and reviews align. |
| **Imports**       | Group: standard library, blank line, third-party, blank line, internal (`goengine/…`). Use `goimports` or editor equivalent.                                                                                                | Readability and consistent grouping.             |
| **Naming**        | Follow Effective Go: short names in small scope; `MixedCaps` for exported; no redundant package name in symbol (e.g. `scene.Scene`, not `scene.SceneManager` if already in `scene`). Acronyms all caps: `ID`, `UI`, `HTTP`. | Idiomatic Go; clear exports.                     |
| **Package names** | Single word, lowercase; no `common` or `util` unless truly minimal. Prefer descriptive names: `physics`, `object`, `event`.                                                                                                 | Discoverability and clear boundaries.            |
| **File length**   | Prefer smaller files; one primary type per file when it keeps the file under ~300 lines. Same package can span files (e.g. `object/gameobject.go`, `object/component.go`).                                                  | Navigability without fragmenting packages.       |
| **Comments**      | Document all exported symbols. Start with the name: `// SceneManager registers scenes and switches the current one.` Use doc comments for packages (`// Package event provides a central event bus.`).                      | Godoc and onboarding.                            |


### 2. Error handling and logging


| Area        | Standard                                                                                                                                                                  | Rationale                              |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| **Errors**  | Return errors; do not panic in library/engine code. Use `errors.Is` / `errors.As` for sentinel and wrapped errors. Define sentinels: `var ErrNotFound = errors.New("…")`. | Predictable control flow; testable.    |
| **Logging** | Use `log` (or an agreed logger) for operational messages. No `fmt.Print` for diagnostics. In hot paths (Update/Draw), avoid logging per frame unless behind a debug flag. | Avoid log spam and allocation in loop. |
| **Context** | Use `context.Context` for request-scoped cancellation and timeouts where we have async or I/O (e.g. resource loading). Not required in every function.                    | Clear cancellation propagation.        |


### 3. Interfaces and dependencies


| Area            | Standard                                                                                                                                                                  | Rationale                                    |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| **Interfaces**  | Prefer small interfaces (1–3 methods). Define in consumer packages or in a shared `ports` package when multiple implementations exist. Accept interfaces, return structs. | Loose coupling; easier testing and swapping. |
| **Constructor** | Use `New`* for constructors. Inject dependencies (event bus, config) via parameters rather than globals.                                                                  | Explicit dependencies; testability.          |
| **No globals**  | Avoid package-level mutable state. Engine, event bus, and config are created in `main`/bootstrap and passed down.                                                         | Determinism and testability.                 |


### 4. Game loop and performance


| Area               | Standard                                                                                                                                                                                            | Rationale                            |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| **Frame budget**   | Assume 60 FPS (~16.67 ms/frame). Keep Update and Draw logic bounded; avoid unbounded loops or heavy allocation in the hot path.                                                                     | Stable frame rate.                   |
| **Allocations**    | Minimize allocations in `Update`/`Draw`: reuse slices, avoid boxing into `interface{}` in hot paths, pool heavy objects if profiling shows need. Do not optimize prematurely; measure first.        | Lower GC pressure; fewer hitches.    |
| **Determinism**    | Where replay or network sync may be required (ADR-003, ADR-004 networking), keep game logic deterministic: avoid `rand` in logic or use a seeded RNG; avoid iteration over maps when order matters. | Reproducibility; sync.               |
| **Float for time** | Use `float64` for delta time and durations in game logic (as we do today). Be consistent with one unit (e.g. seconds).                                                                              | Simplicity; matches Ebiten and math. |


### 5. Project layout and modules


| Area              | Standard                                                                                                                                                                                                            | Rationale                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| **Module**        | Single module `goengine` at repo root. Internal packages under `goengine/…`.                                                                                                                                        | Clear import path; no nested modules.         |
| **Placement**     | Engine core (`engine`, `scene`, `game`), domain packages (`object`, `physics`, `ui`, `event`), and adapters (`physics/box2d`, `resource`) as already present. New features go into the appropriate layer (ADR-003). | Aligns with ADR-001, ADR-003.                 |
| **Config / data** | YAML (or agreed format) for scene/UI data; code for behavior. Keep schema and defaults documented.                                                                                                                  | Data-driven where beneficial; code for rules. |


### 6. Testing


| Area           | Standard                                                                                                                                                   | Rationale                               |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| **Unit tests** | `*_test.go` next to source. Test public API and critical paths. Use table-driven tests where many cases. Mock interfaces (event bus, ports) for isolation. | Regression safety; refactor confidence. |
| **Naming**     | `TestFoo` for function; use `t.Run("scenario", …)` for subtests.                                                                                           | Clear failure identification.           |
| **Game logic** | Prefer testing logic with events (emit intents, assert state or emitted events) rather than full engine bootstrap.                                         | Fast, focused tests.                    |


### 7. Version control and tooling


| Area        | Standard                                                                                                                                 | Rationale                          |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| **CI**      | Run `go build ./...`, `go test ./...`, and a linter (e.g. `staticcheck`, `golangci-lint` with agreed config). Fix warnings before merge. | Consistent quality.                |
| **Linting** | Enable at least: `gofmt`, `go vet`, `staticcheck` (or equivalent). No unused exports, no ignored errors, no unnecessary copies.          | Catch common bugs and style drift. |


## Tradeoffs

### Positive

- **Consistency:** Everyone formats and names the same way; less bike-shedding in review.
- **Onboarding:** New contributors have a single doc to follow; godoc and structure are predictable.
- **Performance awareness:** Standards call out frame budget and allocations so we avoid accidental hot-path waste; determinism is documented for future replay/network work.
- **Testability:** Interfaces and dependency injection (already aligned with ADR-002, ADR-003) are reinforced; event-driven logic is easier to test.
- **Tooling:** gofmt, goimports, and linters enforce much of this automatically.

### Negative

- **Rigidity:** Some rules may feel strict (e.g. no globals, small interfaces). We may need exceptions with justification (e.g. a global debug flag guarded by build tag).
- **Overhead:** Doc comments and tests take time; we accept that for exported API and critical paths, not every internal helper.
- **Performance rules are guidance:** “Minimize allocations” and “determinism” require judgment; we avoid premature optimization but must revisit when profiling or when adding replay/network (ADR-004).

### Alternatives considered

- **Stricter style (e.g. mandatory linter set):** We could lock a specific linter config in CI; for now we recommend a minimal set and leave exact rule set to team agreement.
- **No game-specific rules:** We could rely only on generic Go style; explicitly calling out frame budget and determinism helps avoid mistakes typical in game code.
- **Heavy codegen:** We did not adopt generated code for events or bindings; we keep hand-written Go and optional codegen only if we adopt it later (e.g. for Lua or networking).

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout) (as reference; we follow a flatter layout suited to an engine).
- ADR-001: UI and scene data model.
- ADR-002: Event manager (decoupling, sync delivery).
- ADR-003: Layer separation (Application / Game Logic / Game View; events only across layers).
- Internal: Ebiten game loop; existing packages `object`, `physics`, `scene`, `event`, `ports`.

