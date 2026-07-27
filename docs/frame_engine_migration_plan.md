# Migration Plan: Extracting `frame-engine`

## Vision

Today, `frame` is one Go module containing both the engine (event bus, GameObject/Component
system, physics, scripting, resource loading, rendering plumbing) and the application layer
(`main.go`, the demo games under `games/`). This plan splits that in two:

- **`frame-engine`** ([diego3/frame-engine](https://github.com/diego3/frame-engine)) — an
  importable Go library containing everything that is genuinely engine, with no knowledge of
  any specific game.
- **`frame`** (this repo) — becomes a thin application layer: `main.go`, the WASM entry point,
  and `games/` (demo1, metalslug_demo), depending on `frame-engine` as a normal Go module.

**One-line pitch:** the engine becomes a library; this repo becomes its first consumer.

---

## Why this, why now

- **Reuse.** A second game (or a rewrite of one of the current demos) should be able to `go get`
  the engine instead of forking this whole repo.
- **Forces a real public API boundary.** Right now nothing stops application code from reaching
  into engine internals just because they happen to share a module — extraction forces every
  cross-boundary dependency to go through whatever `frame-engine` actually exports.
- **Forces resolving debt that's been sitting on the books.** Two things called out in this
  repo's own docs turn out to be the same problem this migration has to solve anyway (see below):
  the `// FIXME` in `application/engine/engine.go` about hardcoded scene registration
  (`view/scene/registry.go`'s `Factories["main_menu"]`), and the fact that `view/scene.MainMenu` —
  meant to be the engine's one generic, reusable scene type — has, over the Metal Slug demo's
  build-out, accumulated gameplay logic (hit detection, projectile/sphere spawning, camera
  targeting by hardcoded object names) that has no business living in a general-purpose engine.

---

## Package inventory: what moves, what stays, what needs to split first

| Path | Classification | Notes |
|---|---|---|
| `event/` | **Move** | Generic event bus, no game knowledge. |
| `events/` | **Move** | Generic intent/state event vocabulary (`MoveRequested`, `SceneChanged`, `ScriptEmitted`, `BeginContact`, ...) — none of it is Metal-Slug-specific. |
| `object/` (`GameObject`, `Component`, `Transform`, `Sprite`, `Spritesheet`, `Animator`, `PhysicsBody`, `Script`, `Block`, `Ball`, `IntentBuffer`, `Timer`) | **Move** | Core actor system. |
| `object/enemy.go`, `object/projectile.go` | **Move, but flag it** | Generic enough by name/shape (HP + damage, velocity + damage) to be reusable primitives for *any* 2D action game, not Metal-Slug-specific — but this is a judgment call worth a second opinion (see Open Questions). |
| `physics/`, `physics/box2d/` | **Move** | Box2D wrapper + game-unit abstraction, no game knowledge. |
| `ports/` | **Move** | Shared interfaces (`AssetLoader`, `Scene`, `SceneContext`, `UIRoot`). |
| `process/` | **Move** | Process manager, no game knowledge (currently has zero consumers anywhere — see the `game-architecture` skill, PR #23). |
| `resource/` | **Move** | Asset cache (image/audio/font), `fs.FS` support for WASM. |
| `script/` | **Move** | Lua (`gopher-lua`) and Python (`gpython`) VM integration. |
| `vec2/` | **Move** | Generic 2D vector math. |
| `view/ui/` | **Move** | `Container`, `Button`, `Element` — generic UI widgets. |
| `view/input/` | **Move** | Key-binding manager + intent-emitting adapter, config-driven, no game knowledge. |
| `view/camera/` | **Move** | Generic follow camera, no game knowledge. |
| `application/config/` | **Move** | YAML config schema (window, assets, physics, input, scenes). |
| `application/engine/` | **Move** | DI bootstrap (`engine.New`/`Run`/`Shutdown`), signal handling. |
| `application/game/` | **Move** | `ebiten.Game` implementation (`Update`/`Draw`/`Layout`). |
| `application/data/` | **Move** | Data-driven YAML→GameObject loader + the built-in component builders. |
| `view/scene/manager.go` | **Move** | `SceneManager`, generic scene-switching. |
| `view/scene/physics_system.go` | **Move** | Generic physics step/sync/debug-draw, no game knowledge. |
| `view/scene/aabb_test.go`'s subject (the `aabb`/`aabbOverlap` helpers) | **Move** | Generic rectangle-overlap math — but currently only exists inline inside `mainmenu.go` (see below), not as its own file; needs extracting either way. |
| `view/scene/registry.go` (`Factories` map) | **Redesign, don't just move** | This *is* the documented FIXME. A package-level mutable map is also arguably its own small violation of this project's "No Globals" rule (ADR-006/CLAUDE.md). See Phase 0. |
| `view/scene/mainmenu.go` | **Split — do not move as-is** | See "The blocking design problem" below. This is the one file that cannot simply be copied over. |
| `main.go`, `main_wasm.go`, `build_wasm.sh`, `web/index.html` | **Stay** | Application entry points; rewritten to import `frame-engine`. |
| `games/` (`demo1/`, `metalslug_demo/`) | **Stay** | All game content: config, scenes, scripts, assets. |
| `logic/logic.go` | **Stay (currently empty)** | Doc comment says "Add intent handlers and state types here" — it's a placeholder for *this repo's* game rules, not engine code. |
| `docs/adr/*`, `docs/tdr/*` | **Move (copy), except ADR-011 pending review** | Nearly every existing ADR/TDR documents an engine-internal decision (event bus, layering, testing, object pool, scripting, coding standards, Android/WASM build, metrics, asset loading, scene context, data package). They belong with the code they describe. |
| `docs/game_concept_metal_slug_demo.md`, `docs/game_concept_platformer.md` | **Stay** | Game-specific build plans, not engine documentation. |
| `docs/PYTHON_INTEGRATION.md` | **Move** | Documents the engine's Lua/Python scripting-backend abstraction. |
| `.claude/skills/game-dev`, `.claude/skills/game-architecture` | **Stay, needs a follow-up pass** | Both reference file paths (`object/`, `process/`, `view/camera/`) that will move into the dependency. Not a blocker for the migration itself — flagged as follow-up work once the split lands. |
| `agents/pm/` | **Stay, unrelated** | A Slack PM-agent tool for this project's workflow, not engine or game code. Noted only for completeness. |
| Committed `goengine` binary (repo root, ~20MB) | **Delete, unrelated cleanup** | Looks like an accidental commit (a build artifact), not part of either split target. Worth removing and `.gitignore`-ing regardless of this migration, flagged here since it'll otherwise cause merge noise during the split. |

---

## The blocking design problem: `MainMenu` is no longer generic

`view/scene/mainmenu.go` was originally the engine's one demonstration scene type. Over the
course of building out the Metal Slug demo, it accumulated real gameplay rules that only make
sense for that specific game:

- `spawnEntity`/`spawnProjectile` — generic *mechanism* (Prototype clone), but wired to
  Metal-Slug-specific event names (`"SpawnEntity"`, `"SpawnProjectile"`).
- `updateHitDetection` — Metal-Slug-specific rule: projectile-vs-`Enemy` AABB overlap does
  `enemy.HP -= proj.Damage`.
- `updateProjectiles` — despawns projectiles against `m.levelWidth`/`levelHeight`, config fields
  that only exist because a scrolling level exists.
- `findControlled` — picks "the first GameObject with an `intent_buffer`" — reasonably generic,
  but only because there happens to be exactly one player-controlled entity per scene in the
  current demos; a second demo with different assumptions would need to change this in the
  "engine."
- `cameraTargetCenter` — camera-follow target selection, tied to whatever the demo's player
  object happens to be named.

If `mainmenu.go` were copied into `frame-engine` verbatim, the "generic" library would silently
ship Metal Slug's own gameplay rules — exactly the kind of layering violation ADR-003 (layer
separation) already argues against, just moved from the package boundary to the repo boundary.
**This has to be split before (or as part of) the file move, not after.**

Proposed split:

1. **A generic toolkit stays/moves to `frame-engine`** — the boilerplate every data-driven scene
   needs: script-engine construction and per-object script loading/update dispatch (currently
   `Setup`'s first half and `updateScripts`), physics wiring (already factored out as
   `PhysicsSystem`), the generic Prototype-clone spawn mechanism (`spawnEntity`'s actual cloning
   logic, decoupled from the specific `"SpawnEntity"` event name it currently hardcodes), and the
   AABB helpers as their own reusable, exported functions (they're already effectively generic —
   just inline and unexported today).
2. **A Metal-Slug-specific scene stays in `frame`**, composing that toolkit instead of
   reimplementing it — e.g. `games/metalslug_demo/scene.go` (or a new package under `games/`)
   implementing `ports.Scene` and containing exactly the parts above that are genuinely about
   *this* game: `updateHitDetection`'s damage rule, level-bounds despawn, camera target
   selection.
3. **The scene-registration FIXME gets resolved as a side effect**: instead of a package-level
   `Factories` map inside the engine that has to know every scene type name in advance
   (`"main_menu"` hardcoded today), `frame-engine`'s `scene.Manager` should expose a plain
   `Register(id string, factory SceneFactory)` method (it already does) and let the *application*
   (`frame`'s own `main.go`/bootstrap) register its own scene types by calling it directly —
   removing the global registry entirely rather than relocating it.

This is enough of a design decision that it may deserve its own short ADR once someone starts
implementing it, rather than being decided ad hoc mid-migration.

---

## Proposed target layout

**`frame-engine` (new module, e.g. `github.com/diego3/frame-engine`):**

```
frame-engine/
├── go.mod                          # module github.com/diego3/frame-engine
├── application/{config,engine,game,data}/
├── event/
├── events/
├── object/
├── physics/{,box2d}/
├── ports/
├── process/
├── resource/
├── script/
├── vec2/
├── view/{ui,input,camera}/
├── view/scene/                     # manager.go, physics_system.go, aabb.go, a generic scene toolkit
├── docs/adr/, docs/tdr/            # engine-internal ADRs/TDRs, migrated
└── .github/workflows/test.yml
```

**`frame` (this repo, post-migration):**

```
frame/
├── go.mod                          # requires github.com/diego3/frame-engine
├── main.go, main_wasm.go, build_wasm.sh, web/
├── games/{demo1,metalslug_demo}/
├── games/metalslug_demo/scene.go   # new: the game-specific ports.Scene, built on frame-engine's toolkit
├── logic/                          # this repo's own game-rule placeholder
├── docs/game_concept_*.md
├── .claude/skills/, .agents/skills/
└── agents/pm/
```

---

## Migration phases (build order)

Each phase should be independently verifiable (build + test both repos) before starting the
next, same convention as this project's other build-order plans.

### Phase 0 — Split `MainMenu`, in place, in `frame`

Do the toolkit/gameplay split described above *inside the current single repo* first, with no
new repo involved yet. This is the highest-risk part of the whole migration, and it's much safer
to get right with the existing test suite (`view/scene/manager_test.go`, `aabb_test.go`, the
Python script tests) and a live smoke run available in one place, before also juggling two Go
modules. Resolve the scene-registration FIXME here too.

### Phase 1 — Scaffold `frame-engine`

- `go mod init` with the agreed module path (see Open Questions).
- Copy the packages marked **Move** above, rewriting import paths (`goengine/...` →
  `<module>/...`).
- Bring over `.github/workflows/test.yml`, a `CLAUDE.md` tailored to engine-only conventions
  (most of the current one's content — coding standards, architecture, event bus, components —
  applies verbatim; the Metal Slug/Python-demo-specific "Development Rules" section stays behind
  in `frame`), and the engine-internal ADRs/TDRs.
- Get `frame-engine` building and its tests passing standalone, with no consumer yet — same
  precedent as how `process/` and `vec2/` landed in `frame` itself.

### Phase 2 — Move the split scene toolkit

Move `view/scene/manager.go`, `physics_system.go`, and the newly-extracted generic toolkit +
AABB helpers from Phase 0 into `frame-engine`. Leave the Metal-Slug-specific scene file behind.

### Phase 3 — Wire `frame` to depend on `frame-engine`

- During active co-development, point `frame`'s `go.mod` at `frame-engine` with a `replace`
  directive to a local checkout (both repos will churn together for a while; a `replace` avoids
  needing a tagged release for every iteration). Switch to a real published version once
  `frame-engine` stabilizes.
- Update `main.go`, `main_wasm.go`, and the new `games/metalslug_demo/scene.go` to import
  `frame-engine`'s packages.
- Delete the now-migrated packages from `frame` once it builds and all tests
  (`xvfb-run -a go test ./...`, both scripting backends smoke-tested individually per this
  project's known ebiten-audio-context sandbox limitation) pass against the dependency instead of
  the local copy.

### Phase 4 — Cleanup

- Remove the committed `goengine` binary and add it to `.gitignore`.
- Update `frame`'s `README.md`/`CLAUDE.md` to describe the two-repo architecture.
- Follow-up pass on `.claude/skills/game-dev` and `.claude/skills/game-architecture` — their file
  references now span two repos; not a migration blocker, but should be revisited so they don't
  quietly go stale.

---

## Open questions (need a decision before/at implementation)

- **Module path.** This plan assumes `github.com/diego3/frame-engine`; confirm before running
  `go mod init`.
- **`object.Enemy`/`object.Projectile`: move with the engine, or stay as app-specific?** They're
  generic enough by shape (HP/damage, velocity/damage) to be reusable "genre primitives," but
  they were designed against exactly one game's needs so far. Recommendation above is to move
  them, but this is worth a second look once there's a second real consumer to check the
  assumption against.
- **License/README parity.** `frame` currently has no `LICENSE` file; decide what (if anything)
  `frame-engine` should ship with, independent of this migration.
- **ADR/TDR handling: move or copy?** This plan assumes **move** (delete from `frame` once
  migrated) rather than duplicate-and-diverge, to avoid two copies of the same decision record
  silently drifting apart.
- **Sequencing against in-flight PRs.** As of this plan, the only open PRs (#23, #24) are
  docs-only additions (the `game-architecture` skill and ADR-011) — neither touches code that
  this migration would relocate, so there's currently no rebase-storm risk in starting Phase 0
  right away. Re-check open PRs before starting if time has passed since this plan was written.

---

## Relation to existing debt

- **Directly resolves** the `application/engine/engine.go` FIXME (hardcoded scene registration)
  as a forced side effect of Phase 0, rather than as separate cleanup.
- **Extends ADR-003** (layer separation, currently a package-boundary rule) to the repo boundary.
- **PR #23** (`game-architecture` skill) and **PR #24** (ADR-011, GameObject attachment
  hierarchy) both describe patterns and file paths assuming today's single-repo layout; once this
  migration lands, both need a short follow-up pass to point at the dependency instead — noted
  here so it isn't forgotten, not addressed by this plan itself.
