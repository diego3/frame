# ADR-002: Event manager for decoupling subsystems

## Status

In Progress.

## Context

Subsystems are currently coupled by direct references and closures:

- **Game** implements **SceneSwitcher** and passes itself into **Scene.Setup**; scenes and UI call **SwitchTo(id)** to change scene.
- **MainMenu** reads **input** (e.g. F3) directly to toggle debug draw and orchestrates knight controller, physics, and world in Update.
- **KnightController** reads **input.*** and the knight's **PhysicsBody** / **Animator** directly.
- **UI** button **OnClick** closes over **loader** (and could over **switcher**) to play sound or switch scene.

We want to decouple these so that:

- Scenes and UI do not depend on **SceneSwitcher**, loader, or game loop types.
- Cross-cutting "something happened" flows (scene change, button clicked, collision, debug toggle) can be handled by any subscriber without the emitter knowing who listens.
- Data-driven UI can describe actions declaratively (e.g. emit "scene_change_requested") instead of closing over Go references.

## Decision

### 1. Introduce a central event bus

- A small **event** package provides a **Bus** (or **EventManager**) with:
  - **Emit(event)** – publish that something happened.
  - **Subscribe(eventType, handler)** – register a handler to be called when that event type is published.
- Subsystems depend on the bus instead of on each other for cross-cutting concerns. **Game** (or orchestration) creates the bus and injects it where needed (e.g. into **Scene.Setup**, UI loader, input).

### 2. Event shape

- Prefer **typed structs** per event kind (e.g. **SceneChangeRequested**, **ButtonClicked**, **KeyPressed**) so handlers subscribe by type and get type-safe payloads.
- Alternatively: **string topic + payload** (e.g. `Emit("scene_change", id)`) for simplicity; payload is then often `interface{}` and requires type assertion. Choose based on implementation preference; typed events improve discoverability and refactoring.

### 3. Synchronous delivery

- Handlers are invoked **synchronously** when **Emit** is called (same frame, before **Emit** returns). This fits the game loop and keeps ordering predictable.
- If re-entrancy becomes an issue (e.g. a handler emits again), introduce a small deferred queue: events emitted during delivery are queued and processed after the current **Emit** returns. Do not add async (channels/goroutines) unless there is a clear need.

### 4. Ownership and scope

- **Game** (or engine bootstrap) creates the single bus and subscribes to app-level events (e.g. **SceneChangeRequested** → call **manager.SwitchTo**). Scenes and UI receive the bus (or a narrow interface) in Setup and emit events; they do not receive **SceneSwitcher**.
- Start with **one global bus** for the process. Consider a **per-scene bus** later if scene-local events (e.g. "entity_died") should not leak to other scenes; **Game** can still hold an "app" bus for scene change, quit, etc.

### 5. What to decouple first

| Concern | Change |
|--------|--------|
| Scene change | Scene/UI **emit** e.g. **SceneChangeRequested{SceneID}**. **Game** subscribes and calls **manager.SwitchTo**. Remove **SceneSwitcher** from scene dependencies (or keep it as a thin adapter that emits the event). |
| UI actions | Data-driven UI creates buttons that **emit** e.g. **ButtonClicked{ButtonID}** or **SceneChangeRequested**. Subscribers handle audio, scene change, etc. UI no longer closes over loader or switcher. |
| Debug toggle | Input (or game loop) **emits** e.g. **KeyPressed** or **DebugToggleRequested**. Scene or a debug system subscribes and toggles. Scene does not read input directly for F3. |
| (Later) Gameplay → scene | Gameplay **emits** e.g. **GoalReached** or **LevelComplete**. A subscriber calls scene change. Gameplay stays unaware of scene manager. |
| (Later) Physics | Physics **emits** e.g. **Collision{A, B}**. Gameplay subscribes and may emit higher-level events (e.g. **GoalReached**). Physics stays unaware of game rules. |

Keep **direct calls** for local, same-system flow (e.g. knight controller still calls **Body.SetLinearVelocity**); use **events** for cross-system or "something happened" semantics.

### 6. Minimal first step

- Add an **event** package: **Bus** with **Emit** and **Subscribe** (typed or topic-based, sync).
- **Game** creates the bus, subscribes to **SceneChangeRequested** and calls **manager.SwitchTo**.
- Replace **SceneSwitcher** in **Scene.Setup** with the bus (or an adapter that emits **SceneChangeRequested** so existing scenes keep working). Data-driven UI then emits the same event.
- Leave input, physics, and knight controller unchanged until further decoupling is needed (e.g. F3 → event, collisions → events).

## Consequences

### Positive

- **Decoupling:** Scenes and UI no longer depend on **SceneSwitcher**, loader, or game types for cross-cutting actions. Easier to test (mock bus or assert on emitted events).
- **Extensibility:** New features (analytics, achievements) can subscribe without modifying existing emitters.
- **Data-driven UI:** Buttons can be described in YAML with action types; behavior is implemented by subscribers, so no Go closures in data.

### Negative

- **Indirection:** "Who handles scene change?" is no longer a direct call; trace via subscribers. Mitigate with clear event type names and a single place (e.g. game init) where subscriptions are wired.
- **Ordering:** Multiple handlers for one event type may need a defined order (e.g. registration order or priority) if behavior depends on it.
- **Overuse:** Using events for every small interaction can obscure flow. Reserve events for cross-system or broadcast semantics; keep local control flow as direct calls.

## References

- Internal discussion: event manager for decoupling game engine subsystems.
- Existing code: **Game** (SceneSwitcher), **scene/mainmenu.go**, **ports.Scene** (Setup receives switcher), **ui.Button** (OnClick closure).
