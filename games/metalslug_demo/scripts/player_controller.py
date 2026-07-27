# Player controller script (Python version) — Metal Slug demo, step 3: run + shoot.
# Equivalent to player_controller.lua — attach as the "script" component on the "player" GameObject.
#
# Movement comes from self.get_intent() (IntentBuffer filled by MainMenu on MoveRequested);
# side-scroller, so only move_x is used. Shooting comes from on_event("AttackRequested") (input
# action "attack", same convention as games/demo1's dash/attack), which calls engine.emit with a
# "SpawnProjectile" event; MainMenu (Go) actually builds the projectile GameObject (see
# view/scene/mainmenu.go's spawnProjectile) since self has no position getter and the spawn
# position/size are engine concerns, not script ones.
#
# Projectile movement/off-screen deactivation isn't implemented yet (build order step 4) — a
# spawned projectile is visible but stationary until that lands.
#
# Engine API (injected as module global "engine"):
#   engine.emit(name, payload={})
#
# Entity API (injected as module global "self" before each update call):
#   self.set_velocity(vx, vy)
#   self.get_intent(key) -> float  (keys: "move_x", "move_y")

PLAYER_SPEED = 150

# Facing direction, used as the shot direction; defaults to facing right until the player moves.
facing_x = 1.0

pending_shoot = False


def on_event(name, _payload):
    global pending_shoot
    if name == "AttackRequested":
        pending_shoot = True


def update(dt):
    global facing_x, pending_shoot

    move_x = self.get_intent("move_x")

    if move_x != 0:
        facing_x = 1.0 if move_x > 0 else -1.0

    self.set_velocity(move_x * PLAYER_SPEED, 0)

    if pending_shoot:
        engine.emit("SpawnProjectile", {"dir_x": facing_x, "dir_y": 0})
        pending_shoot = False
