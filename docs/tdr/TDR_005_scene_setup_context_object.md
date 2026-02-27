# TDR-005: Scene Setup context object

## Status

Known

## Context

The `ports.Scene` interface defines a `Setup` method with multiple parameters:

- `Setup(cfg *config.Config, loader ports.AssetLoader, ui ports.UIRoot, switcher ports.SceneSwitcher) error`

This works for now, but as scenes grow and require more dependencies (e.g. logging, debug tools, audio bus, save data), the parameter list may become long and brittle. It also makes it harder to evolve scene dependencies without changing the interface signature in many places.

## Current state

- `ports.Scene` in `ports/ports.go` declares `Setup` with:
  - `*config.Config`
  - `ports.AssetLoader`
  - `ports.UIRoot`
  - `ports.SceneSwitcher`
- `scene.Manager` in `scene/manager.go` calls `Setup` with these arguments when switching scenes.
- Scenes like `scene.MainMenu` implement `Setup` with the same parameter list.

The shape is acceptable today but may not scale well as we introduce more services or cross-cutting concerns that scenes need access to.

## Target state

Encapsulate scene dependencies in a **context object** passed to `Setup`:

- Define a struct (e.g. `SceneContext` or `SceneDeps`) that contains:
  - `Config *config.Config`
  - `Loader ports.AssetLoader` (or smaller loader ports as they are introduced).
  - `UI ports.UIRoot`
  - `Switcher ports.SceneSwitcher`
  - Any future shared services needed by scenes.
- Change `ports.Scene` to:
  - `Setup(ctx *SceneContext) error` (exact name TBD).
- Update `scene.Manager.SwitchTo` to build and pass a `SceneContext` instance instead of multiple parameters.
- Update existing scenes to use fields from the context rather than positional parameters.

This makes it easier to:

- Add new context fields without changing the method signature everywhere.
- Pass optional or feature-specific services in a structured way.
- Keep scene APIs cleaner and better aligned with future growth.

## Effort / priority

- **Effort**: Medium  
  - Requires changing the `Scene` interface, updating manager and scene implementations, and adjusting call sites.
  - Can be done in a focused refactor without changing behavior.
- **Priority**: P2 (medium)  
  - Helpful as soon as more dependencies are added to scenes; good to address before the interface is widely used by many scenes.

