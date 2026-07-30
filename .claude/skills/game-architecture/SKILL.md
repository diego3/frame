---
name: game-architecture
description: Architectural guidance for adding gameplay systems to the goengine (frame) codebase, based on "Game Coding Complete, 4th Edition" (McShaffry & Graham). Use this skill whenever implementing a new gameplay feature under object/, process/, view/camera/, view/scene/, application/data/, or games/*/scripts/ — spawning or pooling entities (projectiles, pickups, explosions), adding timed behavior (cooldowns, delays, camera shake, animation sequences), giving an enemy or actor AI/behavior, wiring a HUD, or deciding whether something belongs in a Component, an Event, or a Process. Also use it when the user asks "how should I structure this", mentions object pooling, prototype/clone, process manager, state machine, or references Game Coding Complete / game architecture patterns directly, even if they don't name a specific file.
---

# Game Architecture (Game Coding Complete patterns)

This engine already implements several architectural patterns from *Game Coding Complete, 4th
Edition* (McShaffry & Graham) — a layered Application/Logic/View split (ADR-003), a type-safe
event bus for decoupled messaging, a GameObject/Component actor system, a resource cache, and a
Prototype-based spawning system. This skill exists so new gameplay code *keeps using* those
patterns instead of re-inventing ad hoc versions of them inside a script or scene file.

It is written for this specific codebase, not as a general game-programming primer. Read
`references/pattern-catalog.md` for the deeper chapter-by-chapter mapping, extended code
sketches, and a worked example (camera shake) once the summary below isn't enough.

## When to Use This Skill

- Adding a new gameplay feature to `games/metalslug_demo/` or any `games/<name>/` demo
- Spawning many short-lived entities (projectiles, explosions, pickups, particles)
- Adding a cooldown, delay, timer, or multi-step timed sequence (attack cooldown, respawn timer,
  explosion animation, camera shake)
- Giving an enemy or actor behavior beyond a single `update(dt)` script function
- Wiring up a HUD or any UI element that needs to reflect gameplay state
- Deciding where a piece of logic belongs: a `Component`, an `event.Bus` message, or a
  `process.Process`
- Reviewing a PR that adds any of the above, to check it isn't bypassing an existing pattern

## Core Concepts

### 1. Respect the layer boundary (Application → Logic → View)

Per `CLAUDE.md` / ADR-003, layers only talk through `event.Bus`. Before adding a field or method
that reaches from `view/scene` into `application/` or vice versa, ask whether an existing event
type in `events/events.go` already covers it, or whether a new one belongs there. Don't smuggle
state across the boundary via a package-level variable or a direct struct reference — that's the
first thing this skill's review pass should flag.

### 2. Actor pattern: Component composition, not inheritance

New gameplay entities (`object.Enemy`, `object.Projectile`, future pickups) are plain structs
implementing `object.Component` (`Type() string`, `Clone() Component`, optionally
`Update`/`Draw`). Compose behavior by attaching multiple small components to one `GameObject`
rather than growing a single component or subclassing. See `object/enemy.go` and
`object/projectile.go` for the current, minimal shape to follow.

### 3. Prototype spawning, not per-frame allocation

`GameObject.IsPrototype` + `Clone(name)` (`object/gameobject.go`) is this engine's object-pool
substitute (Game Coding Complete Ch. 6's actor factory idea, adapted to Go). A prototype
GameObject is defined once in scene YAML (`prototype: true`), never updated or drawn itself
(`object.Manager` skips it), and cloned on demand. **Every TODO that "spawns" something —
pickups, a new projectile type, explosion particles — should clone an existing prototype, not
hand-build a `GameObject` field by field in Go or script code.** `view/scene/mainmenu.go` already
has a *generic* version of this: `spawnEntity` clones any named prototype in response to a
script calling `engine.emit("SpawnEntity", {"prototype": "...", "x": ..., ...})` —
`sphere_timer.py` uses it to spawn `explosion_prototype`, `enemy_bomber.py` and `game_manager.py`
both use it to spawn `sphere_prototype`. Reach for this generic path before adding a new
`spawnX`-style method; `spawnProjectile` predates it and is kept narrow only because the player's
shot has extra facing/clearance logic `spawnEntity`'s payload doesn't cover.

### 4. Three ways to time something — pick by *who owns* the timer

This codebase already has two working, established conventions for multi-frame timed behavior,
both script-side, plus one Go-side mechanism that exists but isn't wired up yet. Pick based on
what the timer belongs to, not by habit:

- **A single-instance script's own module-level state** — e.g. `enemy_bomber.py`'s `_cooldown`,
  `game_manager.py`'s `_next_spawn_in`. Safe *only* because exactly one `GameObject` ever loads
  that script path (`PythonEngine.modules` is keyed by path, not by instance — see
  `script/python_engine.go`). Use this for a unique, named entity's own cooldown.
- **The `object.Timer` component** (`object/timer.go`) — per-instance countdown state attached
  to the `GameObject` itself, read/written via `self.get_timer()`/`self.set_timer()`. Required
  whenever *multiple concurrent clones share the same script path* — `sphere_timer.py` and
  `explosion_effect.py` both use it because several spheres/explosions can be alive at once, and
  a plain Python global would smear their countdowns together. If a new pickup/hazard type will
  ever have more than one instance alive simultaneously, its countdown goes in a `Timer`
  component, not a script global.
- **`process/`** (`Process`, `Base`, `Manager`, `Delay`) — the actual "process manager" from Game
  Coding Complete Ch. 4, for timed behavior that belongs to the *scene/Go layer itself*, with no
  associated scripted `GameObject` to hang a `Timer` component or module global off of.
  **Nothing attaches a `process.Manager` yet** — camera shake is the clean first use case, since
  there's no `GameObject` representing "the camera" to own that state. A scene that needs this
  attaches one `process.Manager` (mirroring how it owns a `*PhysicsSystem`) and calls
  `Update(dt)` on it once per frame; see the Quick Start below. Don't reach for `process/` for
  something that's really an entity's own timer — that's what the two conventions above are for.

### 5. State machines for actor AI, not booleans in a script

Game Coding Complete Ch. 12 recommends explicit finite state machines for actor AI over
scattered flags. `enemy_bomber.py` is already a small, informal instance of this: it tracks
`self.current_animation()` ("idle" vs "attack") as the state, and `update(dt)` is the one place
deciding the transition (`animation_finished()` → back to idle; `_cooldown` expiring → fire and
switch to attack). Follow that shape as enemy AI grows: an explicit named state with one function
deciding transitions, not a growing set of independent booleans (`isAiming`, `hasFired`,
`isDying`, ...) whose combinations become untestable. A plain string-keyed switch is enough at
this scale; don't build a generic FSM framework for it.

### 6. Resource cache — reuse, don't bypass

`resource.Manager` already caches images/audio/fonts by resolved path (`resource/image.go`,
`resource/audio.go`, `resource/font.go`). New sprite/animation work should load through
`ports.ImageLoader`/`loader.LoadImage`, never `ebiten.NewImageFromFile` directly — bypassing the
cache duplicates GPU-side images and breaks WASM's `fs.FS` loading path.

### 7. HUD is View, driven by events — not polling

A HUD element belongs in `view/ui/`, as an `ui.Element` (`Contains`/`HandleClick`/`Draw`) that
subscribes to state events (e.g. a future `EnemyDestroyed`, `PlayerLifeChanged`,
`AmmoChanged` in `events/events.go`) and caches the last value it received — it should not reach
into `object.Manager` or a `GameObject`'s components to read gameplay state directly every
`Draw`. This mirrors how `MainMenuState` already separates what View reads from what Logic owns.

## Quick Start: wiring the Process Manager into a scene

No scene currently attaches a `process.Manager`. This is the shape to add when the first
timed behavior (e.g. camera shake) is implemented:

```go
// In MainMenu (or the relevant Scene): add a field and construct it in Setup.
processes *process.Manager
// ...
m.processes = process.NewManager()

// In Update, alongside the physics step:
m.processes.Update(dt)

// Somewhere an event handler or hit-detection decides "start a shake":
m.processes.Attach(process.NewDelay(0.3)) // placeholder — see references/pattern-catalog.md
                                            // for a real CameraShake process that decays an
                                            // offset over its lifetime instead of just waiting.
```

`process.Delay` alone is just a timer; a `CameraShake` type embeds `process.Base` and does the
actual per-frame work in `Update(dt)`, succeeding when its own duration elapses. The catalog
reference has the full worked example.

## Mapping games/metalslug_demo/ideas.md to patterns

Check current state before assuming a TODO is untouched — several are further along than
`ideas.md`'s checkboxes suggest, and the existing code is the best worked example for the ones
still open.

| TODO | Primary pattern | Existing precedent / where it lives |
|---|---|---|
| Projectile with shader effects | New component + Kage shader (TDR-008) | Not started; `object/`, draw-time in `view/scene` |
| Pickup changes projectile type | Prototype clone (§3) via `spawnEntity`, a new event | Not started; `object/`, `application/data/builders.go`, `events/events.go` |
| Enemy with sprite | `Spritesheet` + `Animator` | Already done for `enemy_bomber` — see `level1.yaml`'s idle/attack strips |
| Enemy shoots at player | Timer/cooldown (§4) + informal state machine (§5) + `spawnEntity` | Already implemented — see `enemy_bomber.py` end to end |
| Enemy dies on collision | AABB overlap + `Enemy.HP` | Implemented for player-projectile-vs-enemy in `updateHitDetection` (`view/scene/mainmenu.go`); player-vs-enemy collision death (the TODO's actual ask) is not |
| Camera shake | `process.Manager` (§4) — the flagship case for it | Not started; `process/`, `view/camera/camera.go` |
| Explosion animation | Non-looping `Spritesheet` + `object.Timer` fallback (§4) | Already done — see `explosion_effect.py` + `explosion_prototype` in `level1.yaml` |
| Initial HUD | Event-driven `ui.Element` (§7), new state events as needed | Not started; `view/ui/`, `events/events.go` |

## Troubleshooting

**I need a cooldown/timer and it feels overkill to define a whole `Process` type for it.**
If it's truly one-shot and nothing else needs to observe it, `process.NewDelay(seconds)` used
standalone is exactly that — no new type needed. Only write a custom `Process` when the timer
also *does* something every frame (like camera shake's decaying offset) or chains into a next
step.

**Spawning pickups/explosions is causing visible GC pauses or frame drops.**
Check whether new `GameObject`s are being built with `object.NewGameObject` + a chain of
`AddComponent` calls at spawn time instead of `prototype.Clone(name)`. Cloning still allocates,
but keeps allocation shape uniform and centralizes the "what does a projectile/pickup look like"
definition in one YAML prototype instead of duplicated Go/script code (see CLAUDE.md's
"minimize allocations in Update/Draw" guidance — spawn logic runs inside `Update`).

**An enemy's behavior is a pile of independent bools that occasionally contradict each other.**
That's the sign to introduce the state machine from §5 — one field holding the current named
state, one function deciding the next state, instead of `isWalking`/`isAiming`/`isFiring` that
can all be true at once by accident.

## Related Skills

- `game-dev` — run this after implementing, to validate tests pass, the golden path still works,
  and no pattern violations were introduced.
