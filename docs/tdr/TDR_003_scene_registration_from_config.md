# TDR-003: Scene registration driven by configuration

## Status

Known

## Context

Scenes are currently registered **directly in code** inside the engine wiring.  
The engine package knows about concrete scene IDs and factories (for example, `"main_menu"`), which couples scene registration to the engine itself.

This is simple for a small prototype, but as we add more scenes (menus, levels, debug tools) it becomes harder to:

- Configure **which scenes exist** and in what order without recompiling.
- Reuse the engine in other games or demos without changing engine code.
- Keep engine responsibilities focused on orchestration rather than game-specific scene lists.

## Current state

- In `engine/engine.go`, the engine constructs a scene manager and hardcodes scene registration:
  - `sm.Register("main_menu", func() (ports.Scene, error) { return scene.NewMainMenu(), nil })`
- The `Game` in `game/game.go` takes an `initialSceneID` (currently `"main_menu"`) and calls `manager.SwitchTo` in `Init`.
- There is no configuration file or data structure that describes the **available scenes** or how they should be wired.

As a result, changing the set of scenes or the initial scene requires editing the engine code instead of updating configuration or game-level bootstrap.

## Target state

Drive scene registration from **configuration or a dedicated bootstrap layer**, not from the engine package:

- Introduce a configuration representation for scenes, for example in `config.Config` or a separate file:
  - A list of scene IDs and (optionally) their factory names or types.
  - A field for the initial scene ID (instead of hardcoding `"main_menu"`).
- Move game-specific scene registrations into:
  - A bootstrap function (e.g. in `main` or a `game_setup` package) that:
    - Parses config.
    - Registers the scenes into the `scene.Manager`.
    - Constructs the `Game` with the configured initial scene ID.
- Keep the engine focused on:
  - Running the loop.
  - Wiring generic dependencies (config, loader, UI, manager).
  - Not on which specific scenes exist in a given game.

This pattern will make it easier to:

- Swap or add scenes without touching engine code.
- Reuse the engine package for other games.
- Potentially drive scene lists from data (e.g. a YAML file) for tools or editors.

## Effort / priority

- **Effort**: Medium  
  - Requires changes to config, bootstrap wiring, and where scenes are registered.
  - Existing behavior can be preserved by keeping `"main_menu"` as the default initial scene when config is missing.
- **Priority**: P2 (medium)  
  - Not urgent for a single-scene prototype, but increasingly valuable as more scenes and flows are added.

