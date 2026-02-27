# TDR-006: Data package load vs build responsibilities

## Status

Known

## Context

The `data` package currently handles several related but distinct concerns:

- Defining the **YAML schema** for scenes (`SceneDef`, `ObjectDef`).
- **Loading** YAML from disk (`LoadScene`).
- **Building** runtime objects from definitions (`BuildWorld`).
- Implementing **all component builders** in a single file (`builders.go`) with direct knowledge of YAML parameter keys.

This is manageable today, but the package is at risk of becoming a catch-all for any data-related logic. As we add more components and data-driven features, the mixture of I/O, schema, and construction logic could make the package hard to navigate and evolve.

## Current state

- `data/scene.go`:
  - Defines `SceneDef` and `ObjectDef`.
  - Implements `LoadScene` (YAML file reading and unmarshalling).
  - Implements `BuildWorld`, which:
    - Iterates objects and components.
    - Uses global `builders` map and the `ComponentBuilder` type to construct components and add them to `object.World`.
- `data/builders.go`:
  - Registers multiple builders in `init()`.
  - Contains all concrete builder implementations for components like `transform`, `sprite`, `spritesheet`, `animator`, `block`, `ball`, `physics_body`.
- `data/params.go`:
  - Provides helper functions for extracting typed values from `map[string]interface{}`.

All of this lives in one package, which is convenient but closely couples disk I/O, schema, and world building.

## Target state

Clarify and, where helpful, separate **load vs build** responsibilities within the `data` package:

- Keep the external API simple (e.g. a function that takes a path and returns a built `object.World`), but internally:
  - Keep YAML schema definitions (structs and tags) clearly grouped.
  - Isolate file I/O and parsing (`LoadScene`) from construction (`BuildWorld`).
  - Organize builders logically (e.g. visual, physics, gameplay) or across multiple files if the list grows large.
- Optionally introduce a small internal layer:
  - A “scene repository” or loader responsible for reading and parsing data.
  - A builder layer responsible solely for translating definitions into runtime objects, using helpers from `params.go`.

The goal is not to over-abstract but to keep the package understandable as it grows and to avoid mixing unrelated responsibilities in single large files.

## Effort / priority

- **Effort**: Low–Medium  
  - Mostly file and function organization, plus possibly a thin internal API or struct to express responsibilities more clearly.
- **Priority**: P3 (low)  
  - Not urgent today but worth addressing incrementally as new components and data-driven features are added.

