# ADR-009: Android Build Approach

## Status

In Review.

## Context

The engine is built on top of [Ebitengine v2](https://ebitengine.org/), which supports Android through the `ebitengine/gomobile` bridge. We need to decide how to produce an Android build (`.apk` / `.aab`) from this Go codebase and how that process fits into the development workflow.

Key concerns:

- **Toolchain complexity**: Android builds require the Android NDK, SDK, and `gomobile`; this is heavier than the desktop build path.
- **Integration with existing code**: Lua scripting (`gopher-lua`), Box2D physics, YAML resource loading, and the event bus all compile to native ARM via CGO; any approach must handle CGO cross-compilation.
- **CI / automation**: Builds should be reproducible and ideally automated.
- **Distribution**: Whether we target sideloading (`.apk`), Google Play (`.aab`), or both.

## Options Considered

### Option A – `gomobile bind` + Android Studio wrapper project

Use `gomobile bind` to compile the engine as an `.aar` library and wrap it in a minimal Android Studio project that calls into the Go library.

**How it works:**
1. Run `gomobile bind -target android ./...` to produce `frame.aar`.
2. Create a thin Android Studio project that imports the `.aar` and calls `EbitenView` from `ebitengine/gomobile`.
3. Build the final `.apk` / `.aab` with Gradle.

**Tradeoffs:**

| | |
|---|---|
| + | Full control over the `AndroidManifest.xml`, permissions, and Gradle configuration. |
| + | Easy to add native Android features (push notifications, in-app purchases, etc.) later. |
| + | Produces a proper `.aab` for Play Store submission. |
| - | Requires maintaining a separate Android Studio project alongside the Go codebase. |
| - | Build steps are split across two toolchains (`gomobile` + Gradle), making CI more complex. |
| - | Developers need Android Studio and a correctly configured NDK version. |
| - | Slower iteration: every Go change requires re-running `gomobile bind` before Gradle picks it up. |

---

### Option B – `gomobile build` (all-in-one APK)

Use `gomobile build -target android` to produce an `.apk` directly from the Go `main` package.

**How it works:**
1. Add the `ebitengine/gomobile` mobile lifecycle import to `main.go` (or a `mobile/main.go` entry point).
2. Run `gomobile build -target android -o frame.apk goengine`.
3. Sign and align the `.apk` with standard Android tooling.

**Tradeoffs:**

| | |
|---|---|
| + | Single command produces a working `.apk`; no Gradle project needed. |
| + | Simplest CI pipeline: install NDK + gomobile, then one build step. |
| + | Keeps all code in one Go repository with no parallel Android project. |
| - | No `.aab` output; Google Play requires `.aab` for new apps since August 2021. |
| - | Very limited control over the manifest, icons, splash screen, and permissions. |
| - | Harder to integrate platform-specific Android features (billing, notifications). |
| - | `gomobile build` wraps `main` with its own activity; the integration point is less flexible than Option A. |

---

### Option C – Docker-based cross-compile via a pre-built Android CI image

Run the build inside a Docker image that pre-installs the Android SDK, NDK, and `gomobile`. This can wrap either Option A or Option B.

**How it works:**
1. Pull or build a Docker image (e.g. `fabernovel/android` or a custom image) with Go, NDK r25+, SDK, and `gomobile` installed.
2. Mount the repo, run `gomobile build` (or `bind` + Gradle) inside the container.
3. Copy the output artifact out of the container.

**Tradeoffs:**

| | |
|---|---|
| + | Hermetic, reproducible builds; no "works on my machine" NDK version issues. |
| + | CI and local builds are identical. |
| + | No Android SDK / NDK installation required on developer machines. |
| - | Large Docker image (Android SDK + NDK can exceed 5 GB). |
| - | Slower first build due to image pull; subsequent builds can be cached. |
| - | Adds Docker as a required dependency for local Android builds. |
| - | Does not solve the `.aab` vs `.apk` question on its own — still needs a choice of Option A or B inside the container. |

---

### Option D – Makefile / shell script orchestration (`gomobile build`) with pinned toolchain versions

A lightweight `Makefile` target that installs the exact required NDK + `gomobile` version and runs `gomobile build`, without Docker.

**How it works:**
1. Add a `Makefile` target `make android` that:
   - Checks for `ANDROID_HOME` / `ANDROID_NDK_HOME`.
   - Installs `gomobile` at a pinned version via `go install`.
   - Runs `gomobile build -target android -o dist/frame.apk ./...`.
2. CI runs `make android` in the same way as developers.

**Tradeoffs:**

| | |
|---|---|
| + | No new tooling (Docker, Gradle) beyond what is already needed for gomobile. |
| + | `Makefile` is already a familiar interface in Go projects. |
| + | Easy to extend with signing, version-bumping, or `aab` steps later. |
| - | Developers must install and configure the Android NDK locally. |
| - | NDK version drift between machines can cause subtle CGO build failures. |
| - | Still produces only `.apk`; switching to `.aab` later requires Option A. |

---

## Decision

**Option B (`gomobile build`) wrapped by Option D (Makefile orchestration)** for the initial implementation.

Rationale:

- The game is not yet targeting Google Play; sideloading or direct distribution is sufficient for the current milestone.
- Keeping the entire build in Go eliminates the need for a parallel Android Studio project.
- A `Makefile` target pins toolchain versions and documents the steps without adding Docker overhead.
- If Play Store submission becomes a requirement, the Makefile can be extended to call `gomobile bind` + Gradle (Option A) with minimal disruption.

The `main.go` entry point will be refactored to expose a `mobile/main.go` build tag variant that satisfies `ebitengine/gomobile`'s lifecycle interface, keeping the desktop entry point unchanged.

## Consequences

### Positive

- Minimal new tooling: one `Makefile` target and a `mobile/main.go` file.
- Desktop and Android share the same codebase and build pipeline.
- CI can add `make android` alongside existing `make test` without major changes.
- CGO dependencies (Box2D, Lua) are handled by `gomobile`'s cross-compile path.

### Negative

- `.apk` only; Google Play submission requires revisiting this decision (move to Option A).
- Developers targeting Android must install the Android NDK (r25 or later) and configure `ANDROID_NDK_HOME`.
- `gopher-lua` and `box2d-go` must be verified to cross-compile cleanly for `arm64` and `arm`; any CGO incompatibilities will surface here.

## References

- [Ebitengine: mobile](https://ebitengine.org/en/documents/mobile.html)
- [gomobile documentation](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile)
- [ebitengine/gomobile](https://github.com/ebitengine/gomobile)
- Existing dependency: `github.com/ebitengine/gomobile v0.0.0-20250923094054-ea854a63cce1` (already in `go.mod`).
