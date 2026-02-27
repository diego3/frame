# TDR-001: Asset loading interfaces and packages

## Status

Known

## Context

The current asset loading is handled by a single concrete type, `resource.Manager`, defined in `resource/resource.go`.  
This manager is responsible for loading and caching **images**, **audio**, **fonts**, and **font faces**, and also exposes a helper for rendering text to images.

While this is simple to start with, it has some downsides:

- The manager mixes **multiple responsibilities** (image, audio, font, and text rendering) into one type and one package.
- The engine code depends on a **fat interface** (`ports.AssetLoader`) instead of small, focused ports.
- Tests and alternative implementations (e.g. headless, editor, or streaming loaders) are harder to introduce, because everything is coupled through one concrete manager.

This technical debt was identified when reviewing the asset-loading code and comparing it with our goals of small interfaces, clear package responsibilities, and ease of swapping implementations.

## Current state

- `resource/resource.go` defines a `Manager` that:
  - Loads and caches images via `LoadImage`.
  - Loads audio bytes and creates players via `LoadAudio` and `NewAudioPlayer`.
  - Loads fonts and exposes `GetFace` for font faces.
  - Provides `TextToImage` as a standalone helper for rendering text.
- `ports.AssetLoader` in `ports/ports.go` is a single interface exposing all of these operations:
  `LoadImage`, `LoadFont`, `GetFace`, `LoadAudio`, `NewAudioPlayer`, and `Clear`.
- Scenes and data builders depend on this **combined** asset loader, even when they only need a subset (for example, sprites only need image loading).

This design works today but makes the resource package a hub for multiple concerns and couples many parts of the engine to a single interface and implementation.

## Target state

Move towards **smaller, focused interfaces and clearer package boundaries** for asset loading:

- Introduce **small ports** in `ports`, such as:
  - `ImageLoader` (image-related methods only).
  - `FontLoader` (font and face methods only).
  - `AudioLoader` (audio loading and player creation).
- Optionally keep a higher-level facade (e.g. an `AssetLoader` or `ResourceProvider`) that **composes** these interfaces but is not required everywhere.
- Split or better organize implementations in `resource`:
  - Either separate loaders into subpackages (`resource/image`, `resource/audio`, `resource/font`) or into well-scoped files/types within `resource`.
  - Keep text rendering helpers (like `TextToImage`) in a place aligned with their responsibility (e.g. font/text rendering) instead of the generic manager.
- Update consumers (scenes, data builders, game/engine wiring) to depend on the **smallest possible port** they need, improving testability and clarity.

With this structure, adding new resource types or swapping implementations (for tests, tools, or different platforms) becomes easier and less invasive.

## Effort / priority

- **Effort**: Medium  
  - Requires touching `ports`, `resource`, and some wiring in `engine`/`game` and scene/data code.
  - Changes are mostly mechanical but need careful interface and dependency updates to avoid breaking the current behavior.
- **Priority**: P2 (medium)  
  - Not blocking current features, but addressing it early will make future asset-related features, tests, and tools cleaner and easier to maintain.

