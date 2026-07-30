# CLAUDE.md — GoEngine (frame)

> AI assistant guide for the `goengine` codebase. Read this before making changes.

---

## Project Overview

**GoEngine** is a modular, event-driven 2D game engine written in Go, built on top of the [Ebiten](https://ebitengine.org/) game library. It provides:

- A layered architecture (Application → Game Logic → View)
- Data-driven scene loading from YAML
- A central type-safe event bus
- Component-based game objects
- Lua 5.1 scripting via `gopher-lua`, and Python 3.4 scripting via `gpython` (the required backend for new game scripts)
- Box2D physics integration

`games/demo1/` and `games/metalslug_demo/` are working example games exercising all major engine systems — `metalslug_demo` is the actively developed one and the default game `main.go` runs.

---

## Repository Structure

The engine (`frameengine/`) and the application layer (`main.go`, `games/`) are split into
separate trees within this one Go module, in preparation for eventually extracting `frameengine/`
into its own importable module (see `docs/frame_engine_migration_plan.md`). **`frameengine/` must
never import from `goengine/games/...`** — dependencies are one-directional (games depend on the
engine, never the reverse); this is checked by grep in code review, not enforced by tooling.

```
.
├── main.go                     # Entry point: loads config, wires scene factories, runs game
├── main_wasm.go                # WASM entry point (demo1 only)
├── go.mod / go.sum             # Go 1.24.0 module (module name: goengine)
├── frameengine/                # The engine — no knowledge of any specific game
│   ├── application/
│   │   ├── config/             # YAML config loading (window, assets, physics, input)
│   │   ├── engine/             # Dependency-injection bootstrap (wires all systems)
│   │   ├── game/                # Game loop: implements ebiten.Game (Update/Draw/Layout)
│   │   └── data/                # Data-driven scene loading (YAML → GameObjects)
│   ├── event/                   # Central event bus + all event type definitions
│   ├── events/                  # Event type definitions (intent/state events)
│   ├── object/                  # GameObject + Component system
│   ├── physics/                 # Physics interfaces
│   │   └── box2d/               # Box2D wrapper (game-unit abstraction)
│   ├── ports/                   # Shared interface definitions (AssetLoader, UIRoot, Scene)
│   ├── process/                 # Process manager (Game Coding Complete Ch. 4 timed behavior)
│   ├── resource/                # Asset caching/loading manager
│   ├── script/                  # Lua + Python (gpython) VM integration
│   └── view/
│       ├── scene/               # SceneManager + the generic WorldScene scene type
│       ├── ui/                  # UI widgets (Container, Button)
│       ├── camera/              # Follow-camera
│       └── input/               # Input adapter (key bindings → intent events)
├── logic/                      # Placeholder for game rule logic
├── games/                      # Application layer: each game is a frameengine consumer
│   ├── demo1/                   # Example game (Lua scripts): config.yaml, scenes/, scripts/, assets/
│   └── metalslug_demo/           # Metal Slug demo (Python scripts); scene.go embeds
│                                 # *scene.WorldScene and adds only this demo's own gameplay
│                                 # rules (shooting, hit detection) — see "Adding a new game" below
├── docs/
│   ├── adr/                    # Architecture Decision Records (ADR-001 through ADR-011)
│   └── tdr/                    # Technical Debt Records
└── scripts/                    # Sample Lua scripts
```

---

## Build & Run

```bash
# Build
go build ./...

# Run the default demo game (games/metalslug_demo/config.yaml)
go run main.go

# Run a specific game's config
go run main.go games/demo1/config.yaml
```

`main.go` defaults to `games/metalslug_demo/config.yaml` but accepts a config path as its first
argument. `main.go` also owns the `sceneFactories` map (scene type name → constructor) passed to
`engine.New` — a new game's scene type must be registered there (or in `main_wasm.go` for the
WASM build) to be usable from its `config.yaml`'s `scenes:` map.

---

## Testing

```bash
# Run all tests
go test ./...

# With race detector (preferred for CI)
go test -race ./...
```

Test files live next to the source they test (`*_test.go`). Current test coverage:
- `frameengine/event/bus_test.go` — event bus delivery, ordering, deferred queue, concurrency
- `frameengine/script/*_test.go` — Lua and Python VM execution, Go callbacks, error handling

### Testing conventions
- **Table-driven tests** with `t.Run()` subtests are preferred.
- Use `testify/assert` for assertions.
- **Mock interfaces**, not concrete types, for isolation.
- Prefer testing via the event bus rather than bootstrapping a full engine.
- Do not add per-frame log calls in tests (frame-budget concern).

---

## Architecture

### Layers

```
Application Layer   │ config, engine bootstrap, game loop (ebiten)
────────────────────┼──────────────────────────────────────────────
Game Logic Layer    │ rules, simulation, physics, scripting
────────────────────┼──────────────────────────────────────────────
View Layer          │ input (intent), scenes, UI, rendering
```

Layers communicate **only via the event bus** — no direct imports across non-adjacent layers.

### Event Bus (`frameengine/event/`)

- Central, synchronous, type-safe dispatch.
- Generic API: `event.Subscribe[T](bus, handler func(T))`, `event.Emit(bus, value)`
- **Deferred queue**: events emitted *during* handler execution are queued and processed after the current dispatch cycle (prevents re-entrancy bugs).
- Thread-safe via `RWMutex`.

**Intent events** (View → Logic): `SceneChangeRequested`, `MoveRequested`, `DebugOverlayToggled`, `QuitRequested`
**State events** (Logic → View): `SceneChanged`, `GameObjectCreated/Destroyed`, `ComponentAdded/Removed`, `ScriptEmitted`

### GameObject & Components (`frameengine/object/`)

- `GameObject`: container with ID, Name, Active flag, and a `map[string]Component`.
- `Component` interface: `Type() string`
- Optional component interfaces: `Updater` (`Update(dt float64)`), `Drawer` (`Draw(*ebiten.Image)`)
- Built-in component types: `Transform`, `Sprite`, `Spritesheet`, `Animator`, `PhysicsBody`, `Script`, `Block`, `Ball`, `IntentBuffer`

### Data-Driven Scenes (`frameengine/application/data/`, `games/demo1/scenes/`)

Scenes are defined in YAML files. A `SceneDef` is a list of `ObjectDef` entries, each with a name and a map of component definitions. Component builders are registered by type name string and convert raw YAML maps into typed component structs.

The engine's one generic scene type is `scene.WorldScene` (`frameengine/view/scene/world_scene.go`
+ `spawn.go`): script/physics/camera wiring, the Prototype-clone `spawnEntity` mechanism, and
projectile movement. A game that needs its own gameplay rules (hit detection, a custom spawn
convention, ...) embeds `*scene.WorldScene` in its own `ports.Scene` implementation rather than
adding that logic to `WorldScene` itself — see `games/metalslug_demo/scene.go` for the pattern
(embeds `WorldScene`, adds only `spawnProjectile` and `updateHitDetection`), and
`docs/frame_engine_migration_plan.md` for why this split exists.

### Scripting (`frameengine/script/`)

- Two backends: pure-Go Lua 5.1 (`gopher-lua`) and pure-Go Python 3.4 (`gpython`).
- **New game scripts must use Python** — see "Development Rules" below; Lua is legacy, kept only
  for `games/demo1`.
- Scripts are attached to `GameObject`s as a `Script` component.
- Engine functions registered in both backends: `play_sound`, `switch_scene`, `quit`, `emit`
- Script entry points called per-frame/per-event: `update(dt)`, `on_event(name, payload)`

### Physics (`frameengine/physics/box2d/`)

- Box2D wrapped with a game-unit abstraction (pixels ↔ Box2D meters via `PixelScale`).
- `BodyDef` for creating bodies (static/kinematic/dynamic, shape, mass, friction, restitution).
- `World.Step(dt)` called once per frame in the physics system.

### Input (`frameengine/view/input/`)

- `Manager`: registry of action → key bindings (loaded from `config.yaml`).
- `Adapter`: polls each frame, emits intent events (`MoveRequested`, `ScriptEmitted`, etc.).
- `ScriptEvents` in config maps action names to Lua event names, enabling fully data-driven input.

---

## Coding Standards (from ADR-006)

### Formatting
- **Always** run `gofmt` / `goimports` before committing.
- Imports ordered: stdlib → blank line → third-party → blank line → internal (`goengine/...`).

### Naming
- Exported identifiers: `MixedCaps`, documented with a comment starting with the name.
- Short names for small scopes; avoid stuttering (`scene.NewManager`, not `scene.NewSceneManager`).
- Packages: single word, lowercase (`physics`, `object`, `event`).

### Interfaces
- Keep interfaces small (1–3 methods).
- Define interfaces in the **consumer** package or in `frameengine/ports/` for shared contracts.
- Accept interfaces, return concrete structs.

### Error Handling
- Return errors; do not `panic` in library code.
- Use `fmt.Errorf("context: %w", err)` for wrapping.
- Use `errors.Is` / `errors.As` for matching.
- Define sentinel errors for stable API contracts.

### No Globals
- No package-level mutable state.
- Config, bus, and engine are created in `main.go` and injected via constructors.

### File Discipline
- Prefer files under 300 lines; one primary type per file.
- One package per directory.

### Performance (game loop)
- Target 60 FPS (~16.67 ms/frame budget).
- **Minimize allocations** in `Update()` and `Draw()` hot paths.
- Do not log per-frame; use debug overlay or profiling tools instead.
- Use `float64` for time/duration (delta seconds).
- Avoid iterating over maps when order matters (non-deterministic).

---

## Development Rules (for Metalslug Demo and beyond)

### Python Scripts Only
- **New scripts must use Python**, not Lua.
- Existing Lua scripts (`scripts/lua/`) are legacy; do not update or maintain them.
- Rationale: Reduce maintenance burden of identical implementations across two languages.

### No Regression Rule
- **Existing game behavior must not break** when adding new features.
- Before committing a feature branch, test the golden path: movement (A/D), shooting (J), jumping (Space), and edge cases.
- If a regression is discovered, fix it in the same commit before pushing.
- Rationale: Silent behavior degradation is worse than incomplete features.

### Unit Tests for Game Scripts
- Python scripts in `games/<game>/scripts/python/` must have unit tests.
- Test files live in the same directory: `player_controller_test.py`, `enemy_walk_test.py`, etc.
- Tests should verify state transitions, velocity calculations, and event handling without running the engine.
- Rationale: Scripts are game logic; they deserve the same test rigor as engine code.
- See `games/metalslug_demo/scripts/python/*_test.py` for examples.

---

## Known Issues / TODOs

- Scene registration is data-driven: each application (`main.go`, `main_wasm.go`) builds its own
  `sceneFactories map[string]scene.SceneFactory` and passes it to `engine.New`; each game's
  `config.yaml` declares `scenes: {id: factory-name}`. The engine itself has no hardcoded scene
  names (this was previously a FIXME; resolved as part of the `frameengine/` consolidation).
- Search for `// TODO`, `// FIXME`, `// HACK` comments in the codebase to find unprioritized debt items.

---

## Adding New Features

| Feature | Where to start |
|---|---|
| New scene | Add YAML in `games/<game>/scenes/`; use `scene.WorldScene` directly if no custom rules are needed, or embed it in a game-specific `ports.Scene` (see `games/metalslug_demo/scene.go`) if they are |
| New component type | Add struct in `frameengine/object/`, register builder in `frameengine/application/data/` |
| New intent/state event | Add type in `frameengine/events/events.go`, subscribe in the appropriate layer |
| New engine API for scripts | Register function in `frameengine/script/` (both Lua and Python backends) |
| New input action | Add key binding in `config.yaml` under `input.keys` |
| New game | Create `games/<name>/` with `config.yaml`, scenes, Python scripts, assets; register its scene factory in `main.go` (or `main_wasm.go`) |

---

## Architecture Decision Records

Key ADRs to read before making structural changes:

| ADR | Topic |
|---|---|
| ADR-003 | Layer separation and event flow — **Implemented** |
| ADR-005 | Lua scripting integration |
| ADR-006 | Coding standards (canonical reference) |
| ADR-001 | UI and scene data model |
| ADR-002 | Event bus design |

ADRs live in `docs/adr/`. When making a significant architectural decision, create a new ADR.
