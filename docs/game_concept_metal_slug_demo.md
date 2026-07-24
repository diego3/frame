# Demo Concept: Metal Slug–style Run-and-Gun

## Vision

A tiny vertical-slice demo proving the engine can drive a Metal Slug–style 2D run-and-gun: a
soldier runs and shoots along a horizontally scrolling level, a camera follows them, and one enemy
type walks toward the player and dies when shot. Scope is deliberately minimal — this is an engine
capability demo, not a game.

**One-line pitch:** Run, shoot, kill one enemy type, on one scrolling level.

---

## Why this, why now

The engine already has the pieces a run-and-gun character needs: component system, multi-animation
sprites (`Spritesheet` + `Animator`), kinematic physics bodies, Lua-scripted behavior, data-driven
YAML scenes. What it doesn't have — camera/scroll, projectiles, and any form of hit
detection/enemy AI — are exactly the gaps already called out in `docs/game_concept_platformer.md`.
Building this demo forces those three features into existence with a concrete, testable target,
instead of speculatively designing them in the abstract.

---

## Scope (MVP)

**Must-have:**

- **Camera follow** — level wider than the screen; camera tracks the player horizontally.
- **Player** — run/idle/shoot animations, moves left/right, shoots on input.
- **Projectile** — spawns at the player, travels in a straight line, is removed on leaving the
  screen or on hitting an enemy.
- **One enemy type** — walks toward the player at a constant speed, has HP, takes damage from
  projectile contact, deactivates on death.
- **One level** — a single static background, ground, and a couple of `Block` placeholders; defined
  in YAML like `games/demo1`.

**Explicitly out of scope for this demo:** parallax layers, multiple enemy types, explosions or
particle effects, ammo/lives/score HUD, vehicles, bosses, level select. Any of these are natural
follow-ups once the MVP proves out, not prerequisites for it.

---

## New engine work required (in build order)

Each step should be independently demoable against the existing `games/demo1` scene before moving
to the next, so a regression is easy to isolate.

### 1. Camera / scroll

Nothing in the engine currently offsets drawing by a world position — `object.Manager.Draw` draws
every `Transform` at its raw coordinates. Needs:

- A `Camera` concept (position, maybe a dead-zone or simple "follow target" reference) — likely
  lives in `view/scene` next to `PhysicsSystem`, or a new `view/camera` package if it grows.
- A draw-time offset applied before `Manager.Draw` (translate the screen, or translate each
  `Transform` read — translating the screen via `ebiten.GeoM` on a sub-image/offscreen is simplest
  and doesn't touch `object.Manager`).
- This is the prerequisite for everything else: without it, the level can't be wider than the
  window, which is the entire visual identity of the genre.

### 2. Hit detection (projectile vs. enemy)

The physics abstraction (`physics.World`/`physics.Body`) has no contact/collision callback today —
`PhysicsSystem` only syncs positions each frame (see `view/scene/physics_system.go`). Two options:

- **(a) Extend `physics.World` with contact events/sensors** — more "correct," but a real change to
  the physics port and the Box2D adapter (`physics/box2d`), i.e. new engine-wide surface area.
- **(b) A lightweight AABB overlap check** — read `Transform` + `Width`/`Height` off projectile and
  enemy `GameObject`s each frame (in Go or Lua) and compare rectangles directly, no physics
  involvement.

**Recommendation: start with (b).** It's a few dozen lines, doesn't touch the physics port, and is
easy to rip out in favor of (a) later if projectile volume or accuracy needs demand it.

### 3. Projectile lifecycle

- New component type (e.g. `object.Projectile`: velocity, damage, lifetime/max distance) registered
  as a data-driven builder in `application/data/builders.go`, or a purely script-driven approach
  (Lua spawns a `GameObject` with `Transform` + `Spritesheet` + a small `update(dt)` that moves it
  and self-deactivates).
- **Removal:** `object.Manager` has no `Remove`; it already skips inactive objects in `Update`/
  `Draw` (see the `AnimatedComponent`/`Active` checks added in the recent tech-debt pass), so
  setting `Active = false` on death or off-screen is enough for the MVP — no need to implement
  actual removal from the slice for this scope.
- **Spawn API:** likely a new `engine.spawn_projectile(x, y, dir)`-style Lua function (same pattern
  as the existing `engine.play_sound`/`engine.emit`), or the player's script directly constructing
  and `Manager.Add`-ing a `GameObject`. Needs a small design decision — see Open Questions.

### 4. Enemy behavior

Simplest possible: a Lua script component (same pattern as `knight_controller.lua`) that walks left
at a constant velocity, with no pathfinding, no state machine. HP and death handled the same way as
the projectile above (`Active = false` on zero HP).

### 5. Data-driven scene wiring

New component types need builders registered (`application/data/builders.go`), and the demo level
itself is a new scene YAML (`games/metalslug_demo/scenes/level1.yaml`) following the exact pattern
`games/demo1/scenes/main_menu.yaml` already establishes.

---

## Suggested build order

1. Camera-follow, validated against the existing `games/demo1` knight scene (no new assets needed).
2. Scaffold `games/metalslug_demo/` (config, one scene, reuse `Block`/`Ball` placeholders for
   background/ground — no new art required yet).
3. Player run + shoot input/script; shooting spawns a projectile `GameObject`.
4. Projectile movement + off-screen deactivation (visible, no enemies yet).
5. Enemy walk script + AABB hit-test vs. projectiles, HP, death.
6. Wire it end-to-end; polish pass (tune speeds, spawn offsets, hit boxes).

Each step lands as its own PR against this plan, same as the recent tech-debt/test PR chain.

---

## Open questions

- **Assets:** no sprite sheets exist for a soldier/enemy yet. Start with `Block`/`Ball` placeholder
  rectangles (matching how `games/demo1` already ships without finished art) and swap in real
  sprites later — don't block the mechanics on art.
- **Where the demo lives:** new `games/metalslug_demo/`, not a modification of `games/demo1` — keeps
  the existing demo stable and the new one disposable/experimental.
- **Projectile spawn API:** Lua-side `engine.spawn_*` function vs. Go-side helper the script calls
  into — worth a quick ADR-style decision once step 3 starts, not before.
- **AABB vs. Box2D contacts:** revisit if the simple overlap check (§2) proves insufficient once
  there's more than one enemy on screen at a time.

## Relation to existing debt

- **ADR-007 (object pool):** flag projectiles as a candidate once this ships and profiling shows GC
  pressure; not needed at MVP scale (a handful of live projectiles at once).
- Unrelated to the still-open TDR-002/006/007 items — this plan doesn't touch those areas.
