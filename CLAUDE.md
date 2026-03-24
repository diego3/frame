# CLAUDE.md — GoEngine (frame)

> AI assistant guide for the `goengine` codebase. Read this before making changes.

---

## Project Overview

**GoEngine** is a modular, event-driven 2D game engine written in Go, built on top of the [Ebiten](https://ebitengine.org/) game library. It provides:

- A layered architecture (Application → Game Logic → View)
- Data-driven scene loading from YAML
- A central type-safe event bus
- Component-based game objects
- Lua 5.1 scripting via `gopher-lua`
- Box2D physics integration

The `games/demo1/` directory contains a working example game that exercises all major engine systems.

---

## Repository Structure

```
.
├── main.go                     # Entry point: loads config, creates engine, runs game
├── go.mod / go.sum             # Go 1.24.0 module (module name: goengine)
├── application/
│   ├── config/                 # YAML config loading (window, assets, physics, input)
│   ├── engine/                 # Dependency-injection bootstrap (wires all systems)
│   ├── game/                   # Game loop: implements ebiten.Game (Update/Draw/Layout)
│   └── data/                   # Data-driven scene loading (YAML → GameObjects)
├── event/                      # Central event bus + all event type definitions
├── object/                     # GameObject + Component system
├── physics/                    # Physics interfaces
│   └── box2d/                  # Box2D wrapper (game-unit abstraction)
├── ports/                      # Shared interface definitions (AssetLoader, UIRoot, Scene)
├── resource/                   # Asset caching/loading manager
├── script/                     # Lua VM integration
├── view/
│   ├── scene/                  # SceneManager + scene implementations (MainMenu)
│   ├── ui/                     # UI widgets (Container, Button)
│   └── input/                  # Input adapter (key bindings → intent events)
├── logic/                      # Placeholder for game rule logic
├── games/
│   └── demo1/                  # Example game: config.yaml, scenes/, scripts/, assets/
├── docs/
│   ├── adr/                    # Architecture Decision Records (ADR-001 through ADR-008)
│   └── tdr/                    # Technical Debt Records
└── scripts/                    # Sample Lua scripts
```

---

## Build & Run

```bash
# Build
go build ./...

# Run the demo game
go run main.go
```

The engine loads its configuration from `games/demo1/config.yaml` (hardcoded in `main.go`).

---

## Testing

```bash
# Run all tests
go test ./...

# With race detector (preferred for CI)
go test -race ./...
```

Test files live next to the source they test (`*_test.go`). Current test coverage:
- `event/bus_test.go` — event bus delivery, ordering, deferred queue, concurrency
- `script/vm_test.go` — Lua VM execution, Go callbacks, error handling

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

### Event Bus (`event/`)

- Central, synchronous, type-safe dispatch.
- Generic API: `event.Subscribe[T](bus, handler func(T))`, `event.Emit(bus, value)`
- **Deferred queue**: events emitted *during* handler execution are queued and processed after the current dispatch cycle (prevents re-entrancy bugs).
- Thread-safe via `RWMutex`.

**Intent events** (View → Logic): `SceneChangeRequested`, `MoveRequested`, `DebugOverlayToggled`, `QuitRequested`
**State events** (Logic → View): `SceneChanged`, `GameObjectCreated/Destroyed`, `ComponentAdded/Removed`, `ScriptEmitted`

### GameObject & Components (`object/`)

- `GameObject`: container with ID, Name, Active flag, and a `map[string]Component`.
- `Component` interface: `Type() string`
- Optional component interfaces: `Updater` (`Update(dt float64)`), `Drawer` (`Draw(*ebiten.Image)`)
- Built-in component types: `Transform`, `Sprite`, `Spritesheet`, `Animator`, `PhysicsBody`, `Script`, `Block`, `Ball`, `IntentBuffer`

### Data-Driven Scenes (`application/data/`, `games/demo1/scenes/`)

Scenes are defined in YAML files. A `SceneDef` is a list of `ObjectDef` entries, each with a name and a map of component definitions. Component builders are registered by type name string and convert raw YAML maps into typed component structs.

### Lua Scripting (`script/`)

- Pure Go VM (`gopher-lua` — Lua 5.1).
- Scripts are attached to `GameObject`s as a `Script` component.
- Engine functions registered in Lua: `play_sound`, `switch_scene`, `quit`, `emit`
- Lua entry points called per-frame/per-event: `update(dt)`, `on_event(name, payload)`

### Physics (`physics/box2d/`)

- Box2D wrapped with a game-unit abstraction (pixels ↔ Box2D meters via `PixelScale`).
- `BodyDef` for creating bodies (static/kinematic/dynamic, shape, mass, friction, restitution).
- `World.Step(dt)` called once per frame in the physics system.

### Input (`view/input/`)

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
- Define interfaces in the **consumer** package or in `ports/` for shared contracts.
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

## Known Issues / TODOs

- **FIXME in `application/engine/engine.go`**: Scene registration is hardcoded (`"main_menu"`). It should be data-driven (tracked, not yet resolved).
- Search for `// TODO`, `// FIXME`, `// HACK` comments in the codebase to find unprioritized debt items.

---

## Adding New Features

| Feature | Where to start |
|---|---|
| New scene | Add YAML in `games/<game>/scenes/`, register in `view/scene/` |
| New component type | Add struct in `object/`, register builder in `application/data/` |
| New intent/state event | Add type in `event/events.go`, subscribe in the appropriate layer |
| New engine API for Lua | Register function in `script/vm.go` via `RegisterEngine` |
| New input action | Add key binding in `config.yaml` under `input.keys` |
| New game | Create `games/<name>/` with `config.yaml`, scenes, scripts, assets |

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
