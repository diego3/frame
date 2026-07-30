# Behavior pattern sketches

Worked examples for `game-ai-behavior`'s core concepts, sized for this engine's actual script API
(`script/python_engine.go`) and component set (`object/`). None of this code exists in the repo
yet — these are sketches for when a concrete `ideas.md`/design TODO needs them, not a description
of current behavior.

## 1. Explicit FSM table (graduating from `enemy_bomber.py`'s informal 2-state version)

`enemy_bomber.py` today is a correct 2-state FSM expressed as a single `if`:

```python
if self.current_animation() == "attack" and self.animation_finished():
    self.play_animation("idle")
```

Once a third state is needed (e.g. "fleeing" when health drops below a threshold), replace the
growing `if`/`elif` chain with a small dispatch table. Each handler reads `self`/`engine` state
and returns the next state name (or `None` to stay):

```python
# enemy_patrol_ai.py — sketch, not implemented.
_state = "idle"

def _idle(dt):
    if engine.get_entity_position("player", "x") is not None and _player_in_range():
        return "attack"
    return None

def _attack(dt):
    if self.animation_finished():
        return "flee" if _low_health() else "idle"
    return None

def _flee(dt):
    self.set_facing(1 if _fleeing_direction() > 0 else -1)
    if _safe_distance_reached():
        return "idle"
    return None

_STATES = {"idle": _idle, "attack": _attack, "flee": _flee}

def update(dt):
    global _state
    next_state = _STATES[_state](dt)
    if next_state is not None and next_state != _state:
        self.play_animation(next_state)
        _state = next_state
```

For **per-instance** state (multiple clones of the same enemy alive at once — e.g. several
patrol enemies), don't use a module-level `_state` global (it would smear across instances the
same way `sphere_timer.py`'s doc comment warns about for countdowns). Store the state index in
an `object.Timer`-adjacent per-instance slot, or — if the engine needs to support this
commonly — that's a signal to add a small typed component (e.g. `object.AIState string`) rather
than overloading `Timer`. Don't build that component speculatively; only once a second AI script
actually needs per-instance state does the shared-module-global problem become real.

## 2. Utility scoring for choosing between actions

For an actor with several candidate actions, score each and pick the highest instead of nesting
conditionals:

```python
# Sketch: enemy chooses between "attack", "retreat", "reposition" each frame.
def _score_attack(dist_to_player, health):
    if dist_to_player > ATTACK_RANGE:
        return 0.0
    return 1.0 - (dist_to_player / ATTACK_RANGE)

def _score_retreat(dist_to_player, health):
    if health > RETREAT_HEALTH_THRESHOLD:
        return 0.0
    return 1.0 - (health / RETREAT_HEALTH_THRESHOLD)

def _score_reposition(dist_to_player, health):
    return 0.3  # low-priority default so the enemy isn't frozen when nothing else scores

def update(dt):
    dist = abs(engine.get_entity_position("player", "x") - self.get_position("x"))
    health = self.get_health()  # hypothetical — actual API depends on the component added

    candidates = {
        "attack": _score_attack(dist, health),
        "retreat": _score_retreat(dist, health),
        "reposition": _score_reposition(dist, health),
    }
    best = max(candidates, key=candidates.get)
    _ACTIONS[best](dt)
```

Prefer moving the tunable constants (`ATTACK_RANGE`, `RETREAT_HEALTH_THRESHOLD`, per-action base
weights) into the prototype's YAML definition and reading them via a script-init hook, the same
way `Spritesheet`/`PhysicsBody` fields are data-driven — this lets a designer retune an enemy's
aggression without touching Python, matching CLAUDE.md's "data-driven" philosophy. If the script
API doesn't yet expose custom per-prototype scalar fields to scripts, that's a small, focused
engine addition (a new field on the `Script` component definition), not a reason to hardcode the
weights.

## 3. A* over a coarse platform grid

This engine's levels are AABB/Box2D-static-body platformer geometry
(`view/scene/world_scene.go`, `object/physics_body.go`), not a 3D navmesh. If an enemy needs to
navigate (e.g. path around a gap instead of walking into it), build the graph from level
geometry at scene load rather than a fine per-pixel grid:

1. At scene setup, walk the level's static `PhysicsBody` platforms and quantize them into coarse
   cells (e.g. one node per platform segment, or a fixed-size grid — tens of nodes for a level
   this engine's size, not thousands).
2. Connect adjacent/overlapping-reachable nodes as graph edges (walkable, or a "jump" edge where
   vertical clearance and gap width are within the actor's jump parameters — reuse
   `player_controller.py`'s `JUMP_IMPULSE`-derived reach, don't invent a second jump model).
3. Run standard A* (Manhattan or Euclidean heuristic — this is 2D, not book's 3D graph distance)
   over that small graph. A plain Python priority-queue A* is fine at this node count; no engine
   API changes needed unless script execution time becomes a measured problem.
4. Cache the computed path per pursuit, not per frame — recomputing A* every `update(dt)` tick
   for a graph that hasn't changed wastes the frame budget (CLAUDE.md: minimize per-frame
   allocation/work).

Do not build this until a concrete TODO needs actual navigation around obstacles.
`enemy_walk.py`'s fixed-bounds patrol has no pathfinding and is the right level of complexity
for "walk along this platform."

**Time-slicing a search that gets expensive.** If a level ever grows enough graph nodes that a
full A* search in one frame is measurable (unlikely at this game's scale, but the technique is
cheap insurance): drive the search from a `process.Process` (`process/`) instead of a plain
function call — expand a bounded number of open-set nodes per `Update(dt)`, and `Succeed()` once
the goal is reached or the open set is exhausted. That's `process.Manager`'s cooperative-
multitasking model (Ch. 4), already built and unwired (see `game-architecture`'s §4), applied to
a search instead of a timer.

## 4. Steering behaviors (`vec2`-based)

Seek (move toward a point), Flee (move away), and Arrive (Seek that decelerates on approach
instead of overshooting) are all the same shape: compute a *desired* velocity, steer the
*current* velocity toward it by a bounded acceleration.

```python
# Sketch: blend Seek-toward-player with Flee-if-too-close, added to enemy_walk.py-style
# movement. Uses only self.get_position/self.get_velocity/self.set_velocity (hypothetical —
# actual script API surface for velocity control may need a small, focused addition; position
# and facing already exist).
MAX_SPEED = 120.0       # game units/s
MAX_ACCEL = 300.0       # game units/s^2
FLEE_RADIUS = 40.0      # too close: back off instead of approaching
ARRIVE_RADIUS = 150.0   # start slowing down within this distance

def _seek(to_x, to_y, from_x, from_y, slow_radius=None):
    dx, dy = to_x - from_x, to_y - from_y
    dist = (dx * dx + dy * dy) ** 0.5
    if dist < 1e-4:
        return 0.0, 0.0
    speed = MAX_SPEED
    if slow_radius is not None and dist < slow_radius:
        speed = MAX_SPEED * (dist / slow_radius)  # Arrive: linear falloff near the target
    return (dx / dist) * speed, (dy / dist) * speed

def update(dt):
    px, py = self.get_position("x"), self.get_position("y")
    tx = engine.get_entity_position("player", "x")
    ty = engine.get_entity_position("player", "y")
    dist = ((tx - px) ** 2 + (ty - py) ** 2) ** 0.5

    if dist < FLEE_RADIUS:
        desired_x, desired_y = _seek(px, py, tx, ty)  # Flee is just Seek with source/target swapped
    else:
        desired_x, desired_y = _seek(tx, ty, px, py, slow_radius=ARRIVE_RADIUS)

    vx, vy = self.get_velocity()
    # Steer current velocity toward desired, clamped to MAX_ACCEL — this is the "steering" part;
    # snapping straight to desired_x/y would look robotic, the same flatness this replaces.
    ax, ay = desired_x - vx, desired_y - vy
    accel_mag = (ax * ax + ay * ay) ** 0.5
    if accel_mag > MAX_ACCEL:
        ax, ay = ax / accel_mag * MAX_ACCEL, ay / accel_mag * MAX_ACCEL
    self.set_velocity(vx + ax * dt, vy + ay * dt)
    self.set_facing(1 if vx >= 0 else -1)
```

Combining more than one behavior (e.g. Seek-the-player + a mild Avoid-the-platform-edge) is a
weighted sum of the desired vectors before the steer-toward-desired step — don't reach for
priority-dithering or truncated-sum schemes built for crowds; a straight weighted average is
plenty at 1-3 enemies on screen.

## 5. Perception gate (no sensory omnipotence)

The minimal version is a distance check before the script *acts* on player position — it doesn't
need to be more than that unless a design explicitly calls for stealth/line-of-sight puzzles:

```python
# Sketch: enemy only "notices" the player within SIGHT_RADIUS, and forgets shortly after losing
# them (a tiny bit of "sensory memory" instead of instantly reacting/forgetting every frame).
SIGHT_RADIUS = 220.0
FORGET_AFTER = 1.5  # seconds of no line-of-sight before giving up

_aware = False
_time_since_seen = 0.0

def update(dt):
    global _aware, _time_since_seen

    px, py = self.get_position("x"), self.get_position("y")
    tx = engine.get_entity_position("player", "x")
    ty = engine.get_entity_position("player", "y")
    dist = ((tx - px) ** 2 + (ty - py) ** 2) ** 0.5

    if dist <= SIGHT_RADIUS:
        _aware = True
        _time_since_seen = 0.0
    elif _aware:
        _time_since_seen += dt
        if _time_since_seen >= FORGET_AFTER:
            _aware = False

    if not _aware:
        return  # stay idle/patrol — no reaction to a player it hasn't perceived

    # ... react to tx, ty as normal (attack, steer toward, etc.) ...
```

For true line-of-sight (not just distance) — e.g. a wall should block detection, not just
range — an AABB raycast against the level's static platform bodies is enough; this codebase
already has AABB overlap testing (`scene.AABB`/`scene.AABBOverlap`,
`view/scene/world_scene.go`) to build a simple ray-vs-AABB check on top of, no new collision
system needed. Only add that if a concrete design (an enemy meant to be snuck past, not just
approached) needs it — the plain distance check above covers "doesn't react from across the
level," which is most of what "no sensory omnipotence" actually buys you here.

**AI Regulators (update-frequency throttling).** Not every enemy needs its full decision logic
re-evaluated every single frame — `enemy_bomber.py`'s cooldown check is already effectively this
(only the countdown itself runs every frame; the actual "decide to fire" branch only does
anything once every `FIRE_INTERVAL` seconds). Generalizing that idea to perception/decision
logic that's more expensive than a cooldown float (e.g. the line-of-sight raycast above) is as
simple as a frame-counter modulo — run the expensive check every N frames, reuse the last result
otherwise — no dedicated "regulator" abstraction needed at this game's enemy count. Movement
(steering) and cheap collision still run every frame; it's specifically the *expensive,
infrequently-changing* decisions that benefit from throttling.
