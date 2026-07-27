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
# is_grounded is derived from a *count* of currently-touching GROUND_OBJECTS, not a plain
# overwritten boolean: the player can touch more than one of them at once (e.g. standing on the
# ground right next to a crate), and a boolean would go permanently False the moment contact with
# just ONE of them ends, even while still resting on another (e.g. brushing a crate's corner while
# walking, then separating from it, while never leaving the ground) -- this was reproducible and
# looked exactly like "jumping randomly stops working" during normal movement.
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
#   self.set_facing(dir_x) -- flips the block's facing marker (visual only, see object.Block)

PLAYER_SPEED = 150
JUMP_IMPULSE = 400  # instantaneous upward velocity change applied on jump, in game units/s

# GameObjects the player can stand on; BeginContact/EndContact against any of these toggle is_grounded.
GROUND_OBJECTS = ("ground", "crate_1", "crate_2", "crate_3")

# Facing direction, used as the shot direction; defaults to facing right until the player moves.
facing_x = 1.0

pending_shoot = False
pending_jump = False

# Number of GROUND_OBJECTS currently in contact with the player; is_grounded is derived from this
# (see module docstring above for why a plain boolean isn't enough). Starts at 1: the player
# spawns resting directly on the ground, matching the very first BeginContact that fires once
# physics catches up.
_ground_contacts = 1
is_grounded = True


def on_event(name, payload):
    global pending_shoot, pending_jump, is_grounded, _ground_contacts
    if name == "AttackRequested":
        pending_shoot = True
    elif name == "JumpRequested":
        if is_grounded:
            pending_jump = True
    elif name == "BeginContact":
        if _touches_ground(payload):
            _ground_contacts += 1
            is_grounded = True
    elif name == "EndContact":
        if _touches_ground(payload):
            _ground_contacts = max(0, _ground_contacts - 1)
            is_grounded = _ground_contacts > 0


def _touches_ground(payload):
    """True if the contact payload is between "player" and any GROUND_OBJECTS entry."""
    name_a = payload.get("GameObjectNameA", "")
    name_b = payload.get("GameObjectNameB", "")
    if name_a == "player" and name_b in GROUND_OBJECTS:
        return True
    if name_b == "player" and name_a in GROUND_OBJECTS:
        return True
    return False


def update(dt):
    global facing_x, pending_shoot, pending_jump

    move_x = self.get_intent("move_x")

    if move_x != 0:
        facing_x = 1.0 if move_x > 0 else -1.0
    self.set_facing(facing_x)

    # Only drive the horizontal component; leave vertical velocity to Box2D (gravity + collision
    # response with the ground/crates run automatically for a dynamic body).
    current_vy = self.get_velocity("y")
    self.set_velocity(move_x * PLAYER_SPEED, current_vy)

    if pending_jump and is_grounded:
        self.apply_linear_impulse_to_center(0, -JUMP_IMPULSE)
        pending_jump = False

    if pending_shoot:
        engine.emit("SpawnProjectile", {"dir_x": facing_x, "dir_y": 0})
        pending_shoot = False
