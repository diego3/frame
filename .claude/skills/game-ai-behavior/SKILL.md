---
name: game-ai-behavior
description: Guidance for designing enemy/actor AI and behavior in the goengine (frame) codebase — finite state machines, decision trees, utility scoring, steering behaviors, sensory/perception gating, and pathfinding — adapted from "Game Coding Complete, 4th Edition" (McShaffry & Graham) Ch. 11-13 to this engine's 2D Python-script actors. Use this skill whenever an enemy or NPC needs more than one behavior, an `if`/`elif` chain in a `*.py` script is starting to grow unreadable, an actor needs to "decide" between several actions (attack vs retreat vs reposition), an enemy needs to move naturally (approach, dodge, flee) instead of a fixed straight line, an enemy reacts to the player with no sight/range check ("sensory omnipotence"), or an enemy needs to navigate around obstacles/platforms. Also use it when the user asks about enemy AI, behavior trees, state machines, utility AI, steering behaviors (seek/flee/arrive/pursuit/evade), perception or line-of-sight, fuzzy logic, or pathfinding for `games/*/scripts/python/`. This skill does not cover engine-level architecture (components, events, process manager, prototypes) — see `game-architecture` for that.
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
data-driven decision table, `vec2`-based steering behaviors, a perception/line-of-sight gate, and
an A* grid adapted to this engine's tile/platform layout) once the summary below isn't enough.

## When to Use This Skill

- An enemy/actor script's `update(dt)` needs more than 2-3 named states
- An actor must choose among several actions based on the situation (attack vs. flee vs.
  reposition), not just react to one trigger
- An enemy needs to move naturally — approach, dodge, flee, avoid the player/an obstacle —
  instead of a fixed straight line or standing still (`enemy_bomber.py` and `enemy_walk.py`
  both do the latter today)
- An enemy reacts to the player unconditionally, regardless of distance (see §6 — this is
  already true of `enemy_bomber.py` and worth knowing before adding a second enemy that does
  the same)
- An enemy needs to move around obstacles or along a path instead of a straight line toward the
  player (no pathfinding exists anywhere in this codebase yet)
- Reviewing a PR that adds enemy AI, to check it isn't reinventing a worse version of §1-6 below,
  or reaching for machinery heavier than this game's scale needs (see "Deliberately Out of
  Scope" below)

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
- **`vec2` (2D vector math) already exists** (`vec2/`) and is the right substrate for steering
  behaviors (§5) — they're arithmetic on a velocity vector, nothing more. No new package needed.
- **Every existing enemy script already has sensory omnipotence**: `enemy_bomber.py` calls
  `engine.get_entity_position("player", ...)` unconditionally, every frame, regardless of
  distance or line of sight — there is no perception/sight-radius concept anywhere in this
  codebase yet. That's not a bug (nothing asked for stealth/ambush behavior so far), but it's
  worth knowing before copying that pattern into a new enemy that's supposed to feel like it
  "notices" the player rather than always knowing exactly where they are — see §6.

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
concrete TODO needs actual navigation. If a search ever gets expensive enough to matter (an
unlikely amount of graph nodes at this game's scale, but worth naming), spread it across frames
by driving it from a `process.Process` — a few node expansions per `Update(dt)` tick, yielding
until the next frame — rather than blocking one frame on a full search; that's exactly what
`process.Manager`'s cooperative-multitasking model (already built, still unwired — see
`game-architecture`'s §4) is for.

### 4. Where AI state lives — don't fight the existing timing conventions

AI state follows the same three-way split `game-architecture`'s skill documents for timers:
single-instance module globals for a uniquely-scripted enemy, `object.Timer` when multiple
clones of the same enemy type can be alive at once (e.g. several bombers on screen), and
`process.Manager` only for AI-adjacent behavior with no owning `GameObject` at all (rare — most
actor AI belongs to the actor's own script). Read `game-architecture`'s §4 before adding a new
timing mechanism for AI state.

### 5. Steering Behaviors — natural movement, cheaper than pathfinding or an FSM

Craig Reynolds' steering behaviors (Seek, Flee, Arrive, Pursuit, Evade, Obstacle/Wall Avoidance)
are just per-frame vector arithmetic on a velocity: compute a desired direction, blend it toward
current velocity by some acceleration limit. `vec2/` already has the vector type and operations
(`Rotate`, normalize, etc. — see the ADR-011 attachment-hierarchy work for the same package used
for transform math) this needs; no new engine plumbing required, just script- or
`PhysicsBody`-level math each `update(dt)`. This is usually a **better first move than an FSM or
A*** for "enemy movement feels flat": `enemy_walk.py`'s fixed-bounds patrol and
`enemy_bomber.py`'s stationary lobbing would both read as more alive with Seek-toward-player (in
range) / Flee-when-too-close blended in, with no new state and no pathfinding at all. When
combining more than one behavior (e.g. Seek the player + Avoid the platform edge), a simple
weighted sum of the desired vectors is enough at this game's enemy count — don't reach for
priority-dithering schemes built for crowds of dozens of agents.

### 6. Perception & Sensory Gating — don't let AI cheat

Every enemy script that calls `engine.get_entity_position("player", ...)` today does so
unconditionally — the enemy always knows exactly where the player is, at any distance, through
any wall. That's fine for `enemy_bomber.py`'s existing ambush-turret role (it's meant to always
be aiming at you once active), but it's the wrong default to copy for an enemy that's supposed
to *notice* the player rather than always track them — an idle patroller that should only react
once the player is close, or a hidden enemy that shouldn't shoot through terrain it can't see
over. The fix is a plain range/line-of-sight check before the script *acts* on the player's
position (not before it's technically able to read it, since the script API always exposes
`get_entity_position` — gating is a script-side convention, not an engine-side restriction):
compute distance (and, if needed, an AABB raycast against level geometry for true line-of-sight)
and only transition out of "idle"/"unaware" once the player is within the enemy's sight radius.
See `references/behavior-patterns.md` for a worked sketch. Skip this for enemies where "always
aware" is the intended design (turrets, alarm-triggered spawns) — it's a design choice per
enemy, not a rule to apply everywhere.

## Deliberately Out of Scope (for a linear 2D shooter at this scale)

*Game Coding Complete*'s AI chapters (and material derived from them) also cover techniques
built for open-world or RTS-scale games with dozens of concurrent, long-lived agents. Metal
Slug's enemies are a handful on screen at a time in a single scrolling corridor. Don't introduce
these speculatively — they solve problems this game doesn't have yet:

- **Hierarchical State Machines / nested states** — earn their cost once a state has meaningful
  sub-states of its own (e.g. "Combat" containing "Aiming"/"Reloading"). A flat FSM (§1) covers
  every enemy in this codebase today.
- **GOAP (Goal-Oriented Action Planning)** — full action-sequence planning is for agents that
  assemble novel plans from a large action library. Utility scoring (§2) covers "pick the best
  of a few known actions" without a planner.
- **Composite/Atomic goal trees** — the same idea as GOAP at a different granularity; same
  verdict.
- **NavMesh / Point-of-Visibility graphs, Influence Maps, terrain/territory analysis** — built
  for open, contestable terrain (who controls this area, where's the safe flank). This game's
  levels are a fixed-direction corridor over platforms; the coarse platform-grid A* in §3 is the
  right-sized equivalent, and there's no "territory" to reason about.
- **LOD AI (reduced update rate/complexity for far-away agents)** — matters when many agents are
  simulated off-screen simultaneously. This game's enemies are camera-culled and few; not worth
  the bookkeeping yet. (The related idea of *not* running every enemy's full decision logic
  every single frame, regardless of distance, is still worth keeping in mind — see the AI
  Regulator note in `references/behavior-patterns.md`.)
- **Fuzzy Logic** — smooths hard cutoffs ("close enough" as a degree, not a threshold). Genuine
  polish, not a capability gap; a plain distance comparison is enough until an enemy's behavior
  visibly needs the nuance.

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

**My enemy just walks straight at/away from the player and it looks robotic.** That's §5 before
anything else — steering behaviors are cheaper than an FSM or pathfinding and are usually the
actual fix for "movement feels flat."

**I'm adding an enemy that should only react once it notices the player, not track them from
anywhere on the level.** That's §6 — gate the reaction behind a range (or line-of-sight) check;
don't copy `enemy_bomber.py`'s always-aware pattern unless "always aware" is this enemy's
intended design too.

## Related Skills

- `game-architecture` — engine-level patterns (components, events, process manager, prototype
  spawning, resource cache, HUD). Read this first; it's the source of truth for how this engine
  implements the mechanisms AI code runs on top of.
- `game-dev` — run after implementing, to validate tests pass, the golden path still works, and
  no pattern violations were introduced.
