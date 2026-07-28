---
name: game-ai-behavior
description: Guidance for designing enemy/actor AI and behavior in the goengine (frame) codebase — finite state machines, decision trees, utility scoring, and pathfinding — adapted from "Game Coding Complete, 4th Edition" (McShaffry & Graham) Ch. 11-13 to this engine's 2D Python-script actors. Use this skill whenever an enemy or NPC needs more than one behavior, an `if`/`elif` chain in a `*.py` script is starting to grow unreadable, an actor needs to "decide" between several actions (attack vs retreat vs reposition), or an enemy needs to navigate around obstacles/platforms instead of moving in a straight line. Also use it when the user asks about enemy AI, behavior trees, state machines, utility AI, fuzzy logic, or pathfinding for `games/*/scripts/python/`. This skill does not cover engine-level architecture (components, events, process manager, prototypes) — see `game-architecture` for that.
---

# Game AI & Behavior (Game Coding Complete Ch. 11-13, adapted to 2D)

This engine has **no AI/behavior framework yet** — every enemy script
(`enemy_bomber.py`, `enemy_walk.py`) is a hand-written `update(dt)` function with its own
ad hoc state (a `current_animation()` string, a cooldown float). That's fine at today's scale
(two enemy types, one behavior each), and this skill is *not* telling you to build a generic
FSM/behavior-tree engine speculatively. It exists so that when the next TODO in
`games/metalslug_demo/ideas.md` needs an enemy to choose between more than one behavior, you
reach for the right pattern from *Game Coding Complete* Ch. 11-13 instead of growing another
pile of booleans — and so you know what already exists here versus what you're introducing for
the first time.

Read `references/behavior-patterns.md` for full worked sketches (a `Timer`-backed FSM script, a
data-driven decision table, and an A* grid adapted to this engine's tile/platform layout) once
the summary below isn't enough.

## When to Use This Skill

- An enemy/actor script's `update(dt)` needs more than 2-3 named states
- An actor must choose among several actions based on the situation (attack vs. flee vs.
  reposition), not just react to one trigger
- An enemy needs to move around obstacles or along a path instead of a straight line toward the
  player (`enemy_bomber.py` and `enemy_walk.py` both currently move in straight lines / stand
  still — no pathfinding exists anywhere in this codebase yet)
- Reviewing a PR that adds enemy AI, to check it isn't reinventing a worse version of §1-3 below

## What This Engine Has Today (ground truth, not the book's assumptions)

Game Coding Complete's AI chapters assume a 3D engine with a scene graph and navmesh. Adapt
accordingly:

- **State tracking**: `enemy_bomber.py` already does an *informal* FSM — one string
  (`self.current_animation()`, "idle"/"attack") is the state, `update(dt)` is the only place
  that changes it. This is correct at 2 states; see §1 for when to graduate it to an explicit
  table.
- **No FSM component, no behavior tree, no blackboard, no navmesh/pathfinding exists.** Don't
  assume any of Ch. 11-13's infrastructure is already wired up — you're building the first
  instance of each pattern below when you need it.
- **Scripts have a narrow, deliberately small API** (`self.get_position`, `self.set_facing`,
  `self.play_animation`, `engine.get_entity_position`, `engine.emit`, `object.Timer` via
  `self.get_timer`/`self.set_timer` — see `script/python_engine.go` and `object/timer.go`).
  AI decisions run inside this API, not by reaching into engine internals from Python.
- **Per CLAUDE.md's Development Rules, new scripts are Python only** — the book's examples (and
  this engine's own legacy scripts in `scripts/lua/`) are Lua; translate patterns, not code.

## Core Concepts

### 1. Finite State Machines — an explicit table once you have 3+ states

Two states as an informal `if` (like `enemy_bomber.py`'s idle/attack) is fine — don't introduce
a framework for that. Once a third state appears (e.g. a "fleeing" or "stunned" state), switch
to an explicit transition table: one dict of `state -> handler function`, each handler returning
the next state name. This is Ch. 11's FSM, sized down to a plain Python dict instead of a class
hierarchy — no new engine API needed, it's a script-side convention. See
`references/behavior-patterns.md` for the full pattern with `object.Timer`-backed state
durations (reuse `object/timer.go`, the same component `sphere_timer.py` already uses for
per-instance countdowns — don't invent a second timer mechanism for AI state durations).

### 2. Decision Trees / Utility Theory — for choosing *between* actions, data-driven

When an actor must pick among several candidate actions (not just transition between states),
Ch. 12 recommends scoring each candidate and picking the best (Utility Theory) rather than a
nested `if/elif` chain that becomes unreadable as more actions are added. In this codebase, keep
the *scores* data-driven where possible — e.g. weights read from the prototype's YAML definition
(`application/data/`, same convention as `Spritesheet`/`PhysicsBody` component fields) rather
than hardcoded in the script — so a designer can tune an enemy's aggression without touching
Python. Only build this once an enemy actually has 3+ competing actions; a single attack-or-idle
enemy doesn't need scoring.

### 3. Pathfinding — A* on a coarse grid, not a navmesh

This engine has no navmesh and its levels are simple platformer geometry
(`view/scene/world_scene.go`'s AABB collision, Box2D static bodies for platforms) — a full
navmesh (Ch. 13) is overkill here. If/when an enemy needs to navigate around obstacles, build a
coarse tile grid from the level's static physics bodies and run A* over that, the same
"simplify the world into a graph" idea the book describes for 3D, just in 2D and much coarser
(platform-level connectivity, not a fine walk mesh). Don't attempt this speculatively —
`enemy_walk.py` and `enemy_bomber.py`'s straight-line/stationary behavior is correct until a
concrete TODO needs actual navigation.

### 4. Where AI state lives — don't fight the existing timing conventions

AI state follows the same three-way split `game-architecture`'s skill documents for timers:
single-instance module globals for a uniquely-scripted enemy, `object.Timer` when multiple
clones of the same enemy type can be alive at once (e.g. several bombers on screen), and
`process.Manager` only for AI-adjacent behavior with no owning `GameObject` at all (rare — most
actor AI belongs to the actor's own script). Read `game-architecture`'s §4 before adding a new
timing mechanism for AI state.

## Correcting a Common Misreading of the Book for This Engine

If you're translating book concepts (or an external skill/prompt derived from the book)
literally, watch for these mismatches with how *this* codebase actually solved the same
problems — described accurately in `game-architecture`, not repeated in full here:

- The book's **GUID-based event messaging** (to avoid a monolithic enum) is solved here instead
  by `event.Subscribe[T]`/`event.Emit` — Go generics keyed by `reflect.Type`, not integer GUIDs
  (`event/bus.go`). Don't introduce a GUID scheme; use the existing typed event bus.
- The book's **XML-defined actor templates** are this engine's **YAML** scene/prototype
  definitions (`application/data/`, `games/*/scenes/*.yaml`) — same idea (data-driven
  construction, no recompile), different serialization format.
- The book emphasizes **Lua** for rapid iteration; this codebase's own Development Rules
  (CLAUDE.md) require **new scripts in Python**, treating `scripts/lua/` as legacy/frozen.
- The book's **Resource Cache** describes LRU eviction; `resource.Manager`
  (`resource/manager.go`) caches by path but has **no eviction policy yet** — it grows
  unbounded. That's a known gap, not something to assume is handled; flag it (e.g. a TDR) rather
  than relying on LRU behavior that isn't implemented.
- The book's **defensive debugging** (Abort/Retry/Ignore error dialogs, minidumps) has no
  equivalent here — there's no crash-report/panic-recovery system in this repo. CI
  (`.github/workflows/test.yml`, `codeql.yml`) covers automated build/test on every push, which
  matches the book's "always have a build" principle, but crash telemetry is unimplemented.

## Troubleshooting

**My enemy's `update(dt)` has grown into 6 nested `if`/`elif` branches checking different
combinations of flags.** That's §1's signal — extract an explicit state table before adding a
7th branch, not after.

**I want the enemy to sometimes attack and sometimes retreat, based on its health and distance
to the player.** That's §2 (utility scoring), not a bigger `if` chain — score each candidate
action, pick the highest, and prefer pulling the weights from YAML over hardcoding them.

**I need an enemy to walk around a platform edge instead of walking off it.** That's §3 — but
confirm the TODO actually needs pathfinding and isn't better solved by `enemy_walk.py`'s
existing pattern (patrol between two fixed X bounds, no pathfinding at all covers most "walk
along this platform" cases).

## Related Skills

- `game-architecture` — engine-level patterns (components, events, process manager, prototype
  spawning, resource cache, HUD). Read this first; it's the source of truth for how this engine
  implements the mechanisms AI code runs on top of.
- `game-dev` — run after implementing, to validate tests pass, the golden path still works, and
  no pattern violations were introduced.
