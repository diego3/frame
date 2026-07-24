# Python Integration & Flexible Scripting Engine

## Summary

This document describes the exploration and implementation of Python scripting support in GoEngine, along with the refactoring that makes the scripting backend flexible — so the engine can run game logic written in either Lua or Python.

---

## Motivation

The engine already had a well-designed Lua scripting layer (see [ADR-005](adr/ADR_005_scripting_lua_integration.md)), but the scripting backend was tightly coupled to the concrete `*script.VM` type. Two goals drove this work:

1. **Python support** — Python is more widely known than Lua. Offering it as an alternative scripting language lowers the barrier for new contributors and makes the engine more approachable.
2. **Flexible engine** — The scripting backend should be swappable via configuration, with no code changes required to switch between Lua and Python.

---

## Design

### `script.Engine` Interface

A new `Engine` interface (`script/engine_interface.go`) abstracts the scripting backend:

```go
type Engine interface {
    DoFile(path string) error
    RegisterEngineAPI(playSound, switchScene, quit, emit)
    CallScriptUpdate(path, funcName string, go_ *object.GameObject, dt float64) error
    CallOnEvent(name string, payload map[string]interface{}) error
    Close()
}
```

Both `LuaEngine` and `PythonEngine` satisfy this interface. Callers (`mainmenu.go`, `game.go`) now depend only on the interface, not on any concrete type.

### Backend Selection

The engine is selected at startup from `cfg.ScriptEngine` (config field `script_engine`):

```yaml
script_engine: "lua"     # default — pure-Go Lua 5.1 via gopher-lua
script_engine: "python"  # pure-Go Python 3.4 via gpython
```

`script.NewEngine(engineType)` acts as the factory:

```go
func NewEngine(engineType string) Engine {
    switch engineType {
    case "python":
        return NewPythonEngine()
    default:
        return NewLuaEngine()
    }
}
```

---

## Lua Backend (`LuaEngine`)

`script/lua_engine.go` wraps the existing `*script.VM` so all prior behaviour is preserved:

- All loaded scripts share a single Lua state.
- The global `self` table is rebound before each `CallScriptUpdate` call.
- `CallOnEvent` delivers events to `on_event(name, payload)` in the shared state.
- `RegisterEngineAPI` registers the `engine` Lua table (`engine.play_sound`, etc.).

**No behaviour change** for games using the default Lua backend.

---

## Python Backend (`PythonEngine`)

`script/python_engine.go` uses [gpython](https://github.com/go-python/gpython) — a pure-Go Python 3.4 interpreter. No C extensions or CGo are needed, consistent with the ADR-005 preference for pure-Go VMs.

### Key design decisions

| Concern | Solution |
|---------|----------|
| Isolation | Each script file gets its own `py.Module`; scripts cannot clobber each other's globals |
| `engine.*` API | Injected as a `*py.Module` into each script's globals before execution |
| `self.*` API | Rebuilt as a `*py.Module` per update call; Go closures capture the entity |
| `on_event` | Iterated across all loaded modules and called if defined |
| Type conversion | `script/python_util.go` provides `goValueToPy` / `pyValueToGo` helpers |

### Entity API from Python

Scripts access the same API as Lua, just with Python syntax:

```python
# self is a module-like object injected into globals before update()
self.set_velocity(vx, vy)
self.play_animation("run")
self.reset_animation("dash")
self.current_animation()     # -> str
self.animation_finished()    # -> bool
self.get_intent("move_x")    # -> float
self.get_name()              # -> str
```

### Engine API from Python

```python
engine.play_sound("assets/click.wav")
engine.switch_scene("main_menu")
engine.quit()
engine.emit("DashRequested", {"speed": 10.0})
```

### Event handling from Python

```python
def on_event(name, payload):
    if name == "DashRequested":
        # payload is a dict-like object: payload["key"]
        ...
```

---

## New Files

| Path | Description |
|------|-------------|
| `script/engine_interface.go` | `Engine` interface + `NewEngine` factory |
| `script/lua_engine.go` | `LuaEngine` wrapping existing Lua VM |
| `script/python_engine.go` | `PythonEngine` using gpython |
| `script/python_util.go` | Go ↔ Python value conversion helpers |
| `games/demo1/scripts/knight_controller.py` | Python port of the knight controller |
| `games/demo1/scripts/knight_gamelogic.py` | Python placeholder for game logic |
| `games/demo1/scenes/main_menu_python.yaml` | Scene referencing Python scripts |
| `games/demo1/config_python.yaml` | Config selecting the Python backend |

---

## Modified Files

| Path | Change |
|------|--------|
| `application/config/config.go` | Added `ScriptEngine string` field (`script_engine` in YAML) |
| `view/scene/mainmenu.go` | Uses `script.Engine` interface; backend selected from config |
| `application/game/game.go` | `scriptVM *script.VM` → `scriptEngine script.Engine`; `ScriptVM()` → `ScriptEngine()` |
| `go.mod` / `go.sum` | Added `github.com/go-python/gpython v0.2.0` |

---

## Running the Python Demo

```bash
# Default (Lua):
go run . games/demo1/config.yaml

# Python backend:
go run . games/demo1/config_python.yaml
```

The knight character's movement, dashing, and attack animations are fully driven by `knight_controller.py`, which is a line-for-line port of the original Lua script.

---

## Limitations & Future Work

- **gpython implements Python 3.4** — some modern Python features (f-strings, walrus operator, etc.) are unavailable.
- **No Python stdlib modules** beyond the ones gpython ships (`math`, `time`, `sys`, `marshal`, `builtins`) are available; the full CPython standard library is not supported.
- **Performance** — gpython runs at roughly 20% of CPython speed; for game logic this is typically not a bottleneck at 60 fps.
- **Mixed-language scenes** — the current design chooses one backend per game config. A future improvement could select the backend per-script based on file extension.
- **Script hot-reload** — not implemented; scenes must be reloaded to pick up script changes.
