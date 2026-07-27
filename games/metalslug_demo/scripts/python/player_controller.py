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
# Physics gravity (configured in config.yaml) is applied by the engine; script tracks velocity_y
# across frames and detects ground contact.
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
GROUND_CHECK_Y = 380  # player y when on ground (460 - 50 height / 2)

# Facing direction, used as the shot direction; defaults to facing right until the player moves.
facing_x = 1.0

pending_shoot = False
pending_jump = False
velocity_y = 0
is_grounded = True


def on_event(name, _payload):
    global pending_shoot, pending_jump
    if name == "AttackRequested":
        pending_shoot = True
    elif name == "JumpRequested":
        pending_jump = True


def update(dt):
    global facing_x, pending_shoot, pending_jump, velocity_y, is_grounded

    move_x = self.get_intent("move_x")

    if move_x != 0:
        facing_x = 1.0 if move_x > 0 else -1.0

    # Apply gravity
    velocity_y = velocity_y + GRAVITY * dt

    # Ground detection: if velocity is moving downward (or stationary) and we're at ground level
    current_y = self.get_position("y")
    is_grounded = (velocity_y >= 0 and current_y >= GROUND_CHECK_Y - 5)

    # Apply jump if requested and grounded
    if pending_jump and is_grounded:
        velocity_y = -JUMP_FORCE
        is_grounded = False
        pending_jump = False

    self.set_velocity(move_x * PLAYER_SPEED, velocity_y)

    if pending_shoot:
        engine.emit("SpawnProjectile", {"dir_x": facing_x, "dir_y": 0})
        pending_shoot = False
