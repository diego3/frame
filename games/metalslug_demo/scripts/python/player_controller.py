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
# Jump is triggered by on_event("JumpRequested") and applies an upward impulse if grounded.
# Vertical velocity is tracked and integrated (gravity) in this script rather than by Box2D,
# because the player is a kinematic body: Box2D never applies gravity or collision response to
# kinematic bodies. Ground contact is reported by the engine via on_event("BeginContact"/
# "EndContact") -- driven by physics/box2d's sensor-based overlap detection registered against
# the player's kinematic body (see physics/box2d/world.go's CreateBody/GetContactsNamesThisFrame)
# -- and is the only source of truth for is_grounded.
#
# Engine API (injected as module global "engine"):
#   engine.emit(name, payload={})
#
# Entity API (injected as module global "self" before each update call):
#   self.set_velocity(vx, vy)
#   self.get_intent(key) -> float  (keys: "move_x", "move_y")
#   self.get_position(axis) -> float  (axis: "x", "y")

PLAYER_SPEED = 150
JUMP_FORCE = 400  # upward impulse in game units/s
GRAVITY = 800  # gravity acceleration in game units/s² (must match config.yaml)

# GameObjects the player can stand on; BeginContact/EndContact against any of these toggle is_grounded.
GROUND_OBJECTS = ("ground", "crate_1", "crate_2", "crate_3")

# Facing direction, used as the shot direction; defaults to facing right until the player moves.
facing_x = 1.0

pending_shoot = False
pending_jump = False
velocity_y = 0
is_grounded = True


def on_event(name, payload):
    global pending_shoot, pending_jump, is_grounded
    if name == "AttackRequested":
        pending_shoot = True
    elif name == "JumpRequested":
        pending_jump = True
    elif name == "BeginContact":
        if _touches_ground(payload):
            is_grounded = True
    elif name == "EndContact":
        if _touches_ground(payload):
            is_grounded = False


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
    global facing_x, pending_shoot, pending_jump, velocity_y, is_grounded

    move_x = self.get_intent("move_x")

    if move_x != 0:
        facing_x = 1.0 if move_x > 0 else -1.0

    # Apply jump if requested and grounded (is_grounded is authoritative here: it is only
    # ever set by BeginContact/EndContact, see on_event above -- kinematic bodies never get
    # real collision response from Box2D, so nothing else can tell us we're on the ground).
    if pending_jump and is_grounded:
        velocity_y = -JUMP_FORCE
        is_grounded = False
        pending_jump = False

    if is_grounded and velocity_y >= 0:
        # Resting on the ground: don't accumulate downward velocity (kinematic bodies are
        # never stopped by collision on their own, so gravity would otherwise integrate
        # forever and the player would sink through the floor).
        velocity_y = 0
    else:
        velocity_y = velocity_y + GRAVITY * dt

    self.set_velocity(move_x * PLAYER_SPEED, velocity_y)

    if pending_shoot:
        engine.emit("SpawnProjectile", {"dir_x": facing_x, "dir_y": 0})
        pending_shoot = False
