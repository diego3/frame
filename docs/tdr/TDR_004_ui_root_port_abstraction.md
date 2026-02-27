# TDR-004: UI root port abstraction for elements

## Status

Known

## Context

The UI root port (`ports.UIRoot`) currently exposes an operation that is specific to a **concrete button type**:

- `AddButton(b *ui.Button)`

This ties the port to the `ui` package and to the concept of a button as the primary or only UI element. As we grow the UI layer (labels, panels, sliders, menus), this shape becomes limiting and encourages adding more concrete methods to the port.

## Current state

- `ports.UIRoot` (in `ports/ports.go`) defines:
  - `Update(layoutWidth, layoutHeight int)`
  - `Draw(screen *ebiten.Image, face font.Face)`
  - `AddButton(b *ui.Button)`
- The concrete implementation `ui.Container` in `ui/container.go` holds:
  - `Buttons []*Button`
  - `Update` and `Draw` that iterate the button slice.
- Scenes such as `scene/mainmenu.go` call `root.AddButton(&ui.Button{...})` directly.

This works today, but the port is tightly bound to the `ui` package and button-specific APIs, which makes introducing new element types awkward and couples scene code directly to the concrete UI implementation.

## Target state

Abstract the UI root around **UI elements**, not just buttons:

- Introduce a small `UIElement` interface (or similar) in the `ui` or `ports` layer that captures what the container needs (e.g. `Update`, `Draw`, hit-testing if required).
- Change `ports.UIRoot` to work with this abstraction, for example:
  - Replace `AddButton(*ui.Button)` with `AddElement(UIElement)` (or similar).
- Have `ui.Button` implement the `UIElement` interface, and update `ui.Container` to manage a collection of `UIElement`s instead of only `*Button`.
- Keep higher-level helpers (like convenience methods to add a button) in the concrete UI package, not in the port.

With this change:

- Scenes depend on the **UI port** and element abstraction, not on a specific `ui.Button` struct.
- Adding new UI controls or containers becomes easier and does not require changing the port interface shape.

## Effort / priority

- **Effort**: Medium  
  - Requires defining the element abstraction, updating `UIRoot`, and refactoring `ui.Container` and call sites.
  - Behavior should remain the same if the interface is designed to mirror current needs.
- **Priority**: P2 (medium)  
  - Not blocking current usage but will greatly reduce friction when expanding the UI system.

