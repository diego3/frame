# TDR-002: Color parameter parsing helpers

## Status

Known

## Context

Several components built from YAML scene data need to parse RGBA color parameters (e.g. `color_r`, `color_g`, `color_b`, `color_a`), clamp them to a valid 0–255 range, and construct `color.RGBA` values.

During review, this logic appeared **duplicated** in more than one builder in `data/builders.go`, with similar anonymous `clamp` functions and manual field extraction. This increases the chance of subtle inconsistencies (e.g. default alpha handling) and makes it harder to change or extend color handling in one place.

## Current state

- `data/builders.go` contains builders such as:
  - `buildBlock`
  - `buildBall`
- Both implement their **own** copy of:
  - Extracting `color_r`, `color_g`, `color_b`, `color_a` via `intParam`.
  - Defining an inline `clamp` function to force values into 0–255.
  - Constructing a `color.RGBA` with some defaulting behavior for `a`.
- There is no shared helper in the `data` package to centralize this logic.

As a result, any change to how we interpret color parameters must be repeated across multiple builders, and it is easy for new builders to copy-paste and diverge.

## Target state

Introduce a **shared helper** in the `data` package for color parsing:

- Add a function such as:
  - `func colorFromParams(params map[string]interface{}) (color.RGBA, bool, error)`
  - It should:
    - Read the `color_*` keys.
    - Apply consistent clamping and defaulting (e.g. default alpha to 255 when any color is set and alpha is missing or invalid).
    - Indicate via a boolean whether a color was explicitly provided (so builders can keep defaults when not).
- Update existing builders (`buildBlock`, `buildBall`, and any future visual components) to use this helper instead of duplicating the logic.
- Keep the helper **local to the `data` package** (not exported beyond what is needed) to avoid leaking YAML-specific parameter parsing details into other layers.

This will provide a single, canonical way to parse colors from scene YAML and reduce duplication in builders.

## Effort / priority

- **Effort**: Low  
  - A small helper plus a couple of call site changes.
  - Minimal risk if covered by simple tests or sample scenes.
- **Priority**: P3 (low)  
  - Not blocking, but a good “Boy Scout” refactor to keep the builders maintainable as we add more visual components.

