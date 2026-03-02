# ADR-003: Layer separation and event-driven communication

## Status

In Progress.

## Context

We want a clear separation of three layers (similar to MVC / common game-engine architecture):

- **Application**: platform, window, main loop, lifecycle. No game rules.
- **Game Logic**: rules, simulation, state. No rendering, no knowledge of input source.
- **Game View(s)**: present state to the user; translate input into *intent* events (e.g. "jump requested"). Can be human (UI + input), AI, demo/replay, or remote (network).

**Key idea:** Game Logic must not know *where* an action came from (button, keyboard, AI, network). That allows changing the application or swapping views without touching game rules. The **Event Manager** is the only communication point between layers: no layer holds direct references to another.

## Decision

### 1. Three layers, one event bus

| Layer | Responsibility | Does not know about |
|-------|----------------|---------------------|
| **Application** | Bootstrap, window, loop, scene manager wiring, shutdown. | Game rules, how input is captured. |
| **Game Logic** | Simulation, state, rules. Consumes *intent* events; emits *state/result* events. | Buttons, keys, AI, network—only sees intents. |
| **Game View(s)** | Emit intents from user/AI/demo/network; subscribe to state events to present. | Other views, application internals. |

The **Event Manager** (central bus) is created by Application and injected into Logic and Views. All cross-layer communication goes through it.

### 2. Intent events vs state events

- **Intent events** (View/Application → Logic): semantic requests, e.g. `JumpRequested`, `MoveRequested`, `SceneChangeRequested`, `QuitRequested`. Logic subscribes; it never sees "key pressed" or "button clicked."
- **State events** (Logic/Application → View): what happened in the game, e.g. `SceneChanged`, `ScoreChanged`, `LevelComplete`, `PlayerDied`. Views subscribe to update what they show.

**Short distinction:** Intent events mean "someone wants this to happen" (requests/commands); state events mean "this has happened in the game" (notifications). Intents drive Logic; state events drive what Views present.

Use **typed structs per event** (see ADR-002). Same event types can be emitted by Human View, AI View, or Demo View; Logic stays source-agnostic.

### 3. Who emits, who subscribes (summary)

| Emitter | Subscriber(s) |
|---------|----------------|
| Application (lifecycle) | Game Logic, Views |
| Game Logic (state/result) | Human View, AI View, Demo View, Application |
| Human / AI / Demo View (intents) | Game Logic, Application (e.g. QuitRequested, SceneChangeRequested) |
| Remote Game Logic (network sync) | Game Logic |

Details (full event list per layer) are documented separately; this ADR establishes the pattern.

## What to implement (by complexity)

### Phase 1 — Foundation (low complexity)

- Add **event** package: `Bus` with `Emit(ev)`, `Subscribe(evType, handler)`, synchronous delivery (ADR-002).
- Define a small set of **intent** and **state** event types (e.g. `SceneChangeRequested`, `SceneChanged`, `QuitRequested`).
- **Application** creates the bus, subscribes to `SceneChangeRequested` and `QuitRequested`, performs scene switch and shutdown. Remove `SceneSwitcher` from scene dependencies; scenes (or UI) emit events instead.
- **Game Logic** (current scene/game controller): subscribe to intents; keep existing Update/Draw structure, but input-driven actions (e.g. debug toggle) come from events, not direct input reads.

**Outcome:** Scene change and quit are event-driven; no layer holds a reference to SceneSwitcher for those flows.

**Status: implemented**

### Phase 2 — Full decoupling (medium complexity)

- **Logic vs View split**: Introduce explicit state types per scene (e.g. `MainMenuState`) that only hold simulation data. Logic updates state in `Update(dt)` from intent events; View only reads state and draws. Optionally separate World’s `Draw` behind a view that receives read-only state.
- **Input → intents**: Human View (or a dedicated input adapter) reads keys/buttons and emits `JumpRequested`, `MoveRequested`, `DebugOverlayToggled`, etc. Game Logic and KnightController no longer read input directly; they react to events.
- **Narrow interfaces**: Pass only `Emitter` / `Subscriber` (or intent-only vs state-only) to each layer so the contract is explicit and testable.

**Outcome:** Application, Game Logic, and Human View are fully decoupled; game rules are independent of input source.

**Status: implemented**

### Phase 3 — Multiple views (higher complexity)

- **AI View**: subscribes to state events, emits same intents as human (JumpRequested, MoveRequested). Logic unchanged.
- **Demo View**: replays or scripts intent events; optionally subscribes to state for recording. Logic unchanged.
- **Remote Game Logic**: subscribes to state sync from Game Logic; emits remote intents or state updates. Logic treats remote input like any other intent source.

**Outcome:** New views (AI, demo, network) can be added without changing Application or Game Logic.

## Tradeoffs

### Positive

- **Independence:** Application can change (e.g. different window/backend) without touching game rules. Views can be swapped (human, AI, demo, remote) without touching Logic.
- **Testability:** Logic can be tested by emitting intent events; no need for real input or UI.
- **Clarity:** New developers see a single rule: "cross-layer communication only via Event Manager; Logic only sees intents."

### Negative

- **Indirection:** Flow is traced via events and subscriptions, not direct calls. Requires a single, clear place (e.g. engine bootstrap) where subscriptions are wired, and good event naming.
- **Overuse risk:** Using events for every tiny interaction obscures flow. Use events for cross-layer or broadcast semantics; keep local, same-system flow as direct calls (ADR-002).
- **Phasing:** Full benefit requires completing at least Phase 2; Phase 1 alone only partially decouples.

## References

- ADR-001: UI and scene data model (Scene, SceneManager, screen/UI/gameplay).
- ADR-002: Event manager (Bus, Emit/Subscribe, sync delivery, ownership).
- Discussion: Application / Game Logic / Game View separation; Event Manager as central communication; intent vs state events; event matrix by layer.
