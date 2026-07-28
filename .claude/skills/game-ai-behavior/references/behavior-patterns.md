# Behavior pattern sketches

Worked examples for `game-ai-behavior`'s three core concepts, sized for this engine's actual
script API (`script/python_engine.go`) and component set (`object/`). None of this code exists
in the repo yet — these are sketches for when a concrete `ideas.md`/design TODO needs them, not
a description of current behavior.

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
