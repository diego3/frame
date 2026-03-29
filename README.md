# Frame — GoEngine

A modular 2D game engine written in Go, built on [Ebitengine](https://ebitengine.org/). Designed for small platform games with a data-driven, component-based architecture and a Lua scripting layer for game logic.

## Features

- **Component system** — Unity-style GameObjects with pluggable components (Transform, Spritesheet, Animator, PhysicsBody, Script, IntentBuffer, Block, Ball)
- **Data-driven scenes** — worlds are defined in YAML; no code changes required to add objects or tweak layouts
- **Lua scripting** — pure-Go Lua 5.1 VM ([gopher-lua](https://github.com/yuin/gopher-lua)); write game logic in `.lua` files, no recompile needed
- **Physics** — Box2D integration via [box2d-go](https://github.com/oliverbestmann/box2d-go) (kinematic, dynamic, and static bodies)
- **Event bus** — type-safe synchronous event bus for decoupled communication between systems
- **Input bindings** — configurable keyboard bindings; input actions emit intent events consumed by scripts
- **Asset loading** — sprites, spritesheets, fonts, and audio (WAV) loaded from the game root directory
- **Flexible config** — all engine settings (window, physics, input, assets) live in a single `config.yaml`

## Demo

The bundled demo (`games/demo1`) features a Knight platformer:

- Move: `A` / `D`
- Dash: `Shift`
- Attack: `J` / `K`
- Debug overlay: `F3`

## Getting Started

**Prerequisites:** Go 1.24+

```bash
git clone https://github.com/diego3/frame.git
cd frame
go run . games/demo1/config.yaml
```

## Project Structure

```
frame/
├── main.go                        # Entry point
├── application/
│   ├── config/                    # Config loader (YAML)
│   ├── data/                      # Scene YAML loader + component builders
│   ├── engine/                    # Engine bootstrap (window, signal handling)
│   └── game/                      # Game orchestrator (scene manager, input, loop)
├── object/                        # GameObject and all components
├── physics/                       # Physics interfaces + Box2D implementation
├── script/                        # Scripting backends (Lua via gopher-lua)
├── event/                         # Type-safe event bus
├── ports/                         # Core interfaces (Scene, AssetLoader, UIRoot)
├── resource/                      # Asset loader (images, fonts, audio)
├── view/
│   ├── input/                     # Keyboard input → intent events
│   ├── scene/                     # Scene implementations (MainMenu, PhysicsSystem)
│   └── ui/                        # UI widgets (Button, Container)
└── games/
    └── demo1/                     # Demo game
        ├── config.yaml            # Window, physics, input, asset config
        ├── scenes/main_menu.yaml  # Data-driven scene definition
        └── scripts/               # Lua game logic scripts
```

## Data-Driven Scenes

Scenes are defined in YAML. Each object lists its components:

```yaml
objects:
  - name: knight
    components:
      - type: transform
        x: 100
        y: 200
      - type: script
        path: "scripts/knight_controller.lua"
      - type: physics_body
        body_type: kinematic
        width: 50
        height: 80
      - type: spritesheet
        name: idle
        image: "assets/knight/_Idle.png"
        frame_width: 120
        frame_height: 80
        cols: 10
        fps: 8
      - type: animator
        current: idle
```

## Scripting

Scripts are Lua files attached to GameObjects via the `script` component. The engine calls `update(dt)` every frame and delivers input events via `on_event(name, payload)`.

```lua
-- Entity API (self)
self.set_velocity(vx, vy)
self.play_animation("run")
self.current_animation()    -- -> string
self.animation_finished()   -- -> bool
self.get_intent("move_x")   -- -> number

-- Engine API
engine.play_sound("assets/click.wav")
engine.switch_scene("main_menu")
engine.quit()
engine.emit("DashRequested", { speed = 10 })
```

Input actions can be wired to script events in `config.yaml`:

```yaml
script_events:
  dash: "DashRequested"
  attack: "AttackRequested"
```

## Configuration

```yaml
window:
  width: 800
  height: 600
  title: "My Game"
  resizable: true

physics:
  gravity_y: 800
  pixel_scale: 64   # pixels per meter

input:
  move_left: A
  move_right: D
  dash: [ShiftLeft, ShiftRight]

assets:
  font_path: "assets/font.ttf"
  scene_path: "scenes/main_menu.yaml"
```

## Architecture Decisions

Design decisions are documented as ADRs in [`docs/adr/`](docs/adr/):

- **ADR-005** — Scripting: pure-Go Lua VM (gopher-lua), no CGo
- **ADR-006** — Coding standards
- **ADR-008** — Testing approaches

## Dependencies

| Package | Purpose |
|---------|---------|
| [ebitengine/ebiten](https://github.com/hajimehoshi/ebiten) | 2D rendering, window, audio |
| [yuin/gopher-lua](https://github.com/yuin/gopher-lua) | Pure-Go Lua 5.1 VM |
| [oliverbestmann/box2d-go](https://github.com/oliverbestmann/box2d-go) | Box2D physics |
| [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config parsing |

## License

See [LICENSE](LICENSE).
