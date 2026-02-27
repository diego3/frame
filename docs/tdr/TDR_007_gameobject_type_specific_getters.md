# TDR-007: GameObject type-specific getters

## Status

Known

## Context

`object.GameObject` is the central container for components. It already supports a generic access method:

- `GetComponent(typeName string) Component`

In addition, it defines **type-specific helper methods** such as `Transform()`, `Animator()`, and `PhysicsBody()`. These are convenient but couple `GameObject` directly to specific component types and naming conventions. As we introduce more components, there is a risk of adding more such helpers and gradually bloating the API surface with per-component methods.

## Current state

- `object/gameobject.go`:
  - Stores components in a `map[string]Component`.
  - Provides:
    - `AddComponent(c Component)`
    - `GetComponent(typeName string) Component`
    - Convenience getters:
      - `Transform() *Transform`
      - `Animator() *Animator`
      - `PhysicsBody() *PhysicsBody`
- These helpers:
  - Hardcode the component type strings (e.g. `"transform"`, `"animator"`, `"physics_body"`).
  - Assume knowledge of concrete component types and their Go structs.

While this is fine for a small, fixed set of core components, extending it to many components would make the `GameObject` API large and harder to maintain.

## Target state

Clarify intended usage of `GameObject` access and limit proliferation of type-specific getters:

- Prefer using `GetComponent(typeName)` at call sites, followed by type assertions where needed.
- Keep **only a minimal set** of convenience getters for the most universally used core components (e.g. `Transform()`), and avoid adding new ones for every component type.
- Optionally introduce small helper functions or methods **outside** of `GameObject` for more complex access patterns (e.g. animation helpers), rather than making `GameObject` itself aware of every component.

This keeps `GameObject` focused on being a generic component container, while still allowing ergonomic access where it brings clear value.

## Effort / priority

- **Effort**: Low  
  - Primarily a matter of coding conventions and possibly a small cleanup to remove or consolidate less-used getters.
- **Priority**: P3 (low)  
  - Not urgent and does not block current development, but worth keeping in mind to prevent gradual API bloat as more components are added.

