# Player controller script (Python version) — Metal Slug demo, step 3: run + shoot; enhanced with jump.
# Equivalent to player_controller.lua — attach as the "script" component on the "player" GameObject.
#
# Movement comes from self.get_intent() (IntentBuffer filled by MainMenu on MoveRequested);
# side-scroller, so only move_x is used. Shooting comes from on_event("AttackRequested") (input
# action "attack", same convention as games/demo1's dash/attack), which calls engine.emit with a
# "SpawnProjectile" event; MainMenu (Go) actually builds the projectile GameObject (see
# view/scene/mainmenu.go's spawnProjectile) since self has no position getter and the spawn
# position/size are engine concerns, not script ones.
#
# The player is a DYNAMIC body with fixed_rotation (see level1.yaml) -- the standard Box2D
# character-controller setup: gravity and vertical collision response (landing on the ground/
# crates) are handled entirely by Box2D itself, exactly like any other dynamic body. This script
# only ever sets the horizontal velocity component directly and leaves the vertical component
# alone, except for jumping, which applies an instant upward impulse instead of teleporting the
# velocity. Fixed rotation stops the box from tipping over from asymmetric contact torque (e.g.
# clipping a crate corner), which a free-rotating dynamic body would otherwise do.
#
# Ground contact is reported by the engine via on_event("BeginContact"/"EndContact") -- ordinary
# Box2D solid-body contacts fire correctly here because dynamic-vs-static (and dynamic-vs-
# kinematic) always generate them, unlike kinematic-vs-static which Box2D's collision matrix
# excludes entirely. is_grounded only gates whether a jump is allowed; it does not affect gravity
# or landing, both of which Box2D already handles correctly on its own.
#
# is_grounded is derived from _touching, a {name: currently_touching_bool} map over
# GROUND_OBJECTS, not a plain overwritten boolean or a raw counter:
# - Not a boolean: the player can touch more than one of them at once (e.g. standing on the
#   ground right next to a crate), and a boolean would go permanently False the moment contact
#   with just ONE of them ends, even while still resting on another (e.g. brushing a crate's
#   corner while walking, then separating from it, while never leaving the ground) -- this was
#   reproducible and looked exactly like "jumping randomly stops working" during normal movement.
# - Not a plain increment/decrement counter: the player spawns already resting on "ground", so
#   _touching starts pre-populated with it -- but Box2D still fires its own real BeginContact for
#   that same contact once physics starts stepping, which a counter would count a second time
#   (spawn assumption + real event = 2), so the one real EndContact that eventually follows would
#   only bring it down to 1, never to 0 -- permanently stuck grounded (silently allowing repeated
#   mid-air jumps, since is_grounded never went False).
# - A dict of booleans, not a set: gpython (this engine's Python backend) implements Python's set
#   type with "add" only -- no "discard"/"remove" -- so a set can accumulate names but never drop
#   one; a real EndContact would then raise AttributeError inside on_event, which the engine calls
#   as "_ = engine.CallOnEvent(...)" (see MainMenu), silently discarding the error and leaving
#   is_grounded permanently stuck True. Overwriting a dict entry (_touching[name] = False) needs
#   no removal method at all, so it doesn't hit that gap.
#
# A jump request is also only ever honored at the instant it's grounded -- it is intentionally NOT
# queued for whenever the player next lands. Queuing caused an unrelated but similarly confusing
# symptom: pressing space while airborne appeared to do nothing, then triggered an unrequested
# jump the moment the player touched down.
#
# Engine API (injected as module global "engine"):
#   engine.emit(name, payload={})
#
# Entity API (injected as module global "self" before each update call):
#   self.set_velocity(vx, vy)
#   self.get_velocity(axis) -> float  (axis: "x", "y")
#   self.apply_linear_impulse_to_center(ix, iy)
#   self.get_intent(key) -> float  (keys: "move_x", "move_y")
#   self.get_position(axis) -> float  (axis: "x", "y")
#   self.set_facing(dir_x) -- flips the sprite/block horizontally (visual only)
#   self.play_animation(name) -- switch the visible spritesheet ("idle"/"run"/"jump")
#   self.reset_animation(name) -- restart a one-shot animation from frame 0

PLAYER_SPEED = 150
JUMP_IMPULSE = 400  # instantaneous upward velocity change applied on jump, in game units/s

# GameObjects the player can stand on; BeginContact/EndContact against any of these toggle
# is_grounded. Must list every standable static body in level1.yaml by name -- platform_1..4 were
# missing here originally, so landing on one never set is_grounded, leaving the player stuck
# showing the "jump" animation (and unable to jump again) despite visibly resting on a platform.
GROUND_OBJECTS = (
    "ground", "crate_1", "crate_2", "crate_3",
    "platform_1", "platform_2", "platform_3", "platform_4",
)

# Facing direction, used as the shot direction; defaults to facing right until the player moves.
facing_x = 1.0

pending_shoot = False
pending_jump = False

# {GROUND_OBJECTS name: currently touching?}; is_grounded is derived from this (see module
# docstring above for why neither a plain boolean, a counter, nor a set is enough). Starts with
# "ground" already True: the player spawns resting directly on it, before physics has run even
# once to report the real contact.
_touching = {"ground": True}
is_grounded = True


def on_event(name, payload):
    global pending_shoot, pending_jump, is_grounded
    if name == "AttackRequested":
        pending_shoot = True
    elif name == "JumpRequested":
        if is_grounded:
            pending_jump = True
    elif name == "BeginContact":
        other = _ground_object_touched(payload)
        if other is not None:
            _touching[other] = True
            is_grounded = True
    elif name == "EndContact":
        other = _ground_object_touched(payload)
        if other is not None:
            _touching[other] = False
            is_grounded = any(_touching.values())


def _ground_object_touched(payload):
    """Returns the GROUND_OBJECTS name involved in this player contact, or None."""
    name_a = payload.get("GameObjectNameA", "")
    name_b = payload.get("GameObjectNameB", "")
    if name_a == "player" and name_b in GROUND_OBJECTS:
        return name_b
    if name_b == "player" and name_a in GROUND_OBJECTS:
        return name_a
    return None


def update(dt):
    global facing_x, pending_shoot, pending_jump

    move_x = self.get_intent("move_x")

    if move_x != 0:
        facing_x = 1.0 if move_x > 0 else -1.0
    self.set_facing(facing_x)

    # Animation follows movement state alone (not shooting -- shooting is instantaneous, not a
    # held pose): airborne always shows "jump" (the one-shot animation holds on its last frame,
    # see level1.yaml's loop: false), otherwise "run" while moving or "idle" at rest.
    if not is_grounded:
        self.play_animation("jump")
    elif move_x != 0:
        self.play_animation("run")
    else:
        self.play_animation("idle")

    # Only drive the horizontal component; leave vertical velocity to Box2D (gravity + collision
    # response with the ground/crates run automatically for a dynamic body).
    current_vy = self.get_velocity("y")
    self.set_velocity(move_x * PLAYER_SPEED, current_vy)

    if pending_jump and is_grounded:
        self.apply_linear_impulse_to_center(0, -JUMP_IMPULSE)
        self.reset_animation("jump")  # replay from frame 0 (crouch) instead of resuming mid-air
        pending_jump = False

    if pending_shoot:
        engine.emit("SpawnProjectile", {"dir_x": facing_x, "dir_y": 0})
        pending_shoot = False
