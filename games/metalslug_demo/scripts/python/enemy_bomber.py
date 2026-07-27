# Enemy bomber script (Python) — Metal Slug demo. Attach as the "script" component on a
# stationary enemy (see level1.yaml's "enemy_bomber", standing on platform_1) that periodically
# lobs a bomb toward the player in a parabolic arc.
#
# The bomb itself is the exact same "sphere_prototype" already wired up for the falling-sphere
# hazard (see sphere_timer.py, which owns the countdown/explode/spawn-VFX behavior for every
# sphere regardless of how it got there) -- this script's only job is deciding *when* to throw and
# *with what launch velocity*, a game rule specific to this enemy, per this project's convention of
# keeping such decisions in Python and MainMenu.spawnEntity generic (see view/scene/mainmenu.go).
#
# The parabola is plain projectile motion (SUVAT): pick a fixed flight time T, then solve for the
# constant launch velocity that lands exactly on the target point at t=T (Y-down, gravity
# accelerates downward):
#   x(T) = x0 + vx*T                    -> vx = (x1-x0)/T
#   y(T) = y0 + vy*T + 0.5*GRAVITY*T^2  -> vy = (y1-y0-0.5*GRAVITY*T^2)/T
# The target's y is the player's *feet*, not its Transform (top-left of its physics body) --
# aiming at the raw Transform.y would compute a velocity that "lands" partway up the player's own
# height, well before the bomb's real collision with the ground, so it keeps falling/travelling
# past that point and overshoots horizontally. PLAYER_FEET_OFFSET mirrors the player's own
# physics_body offset_y + height (20 + 60, see level1.yaml's "player") the same way
# player_controller.py hardcodes JUMP_IMPULSE -- scripts have no API to read another entity's
# component data, only its Transform position.
# GRAVITY mirrors games/metalslug_demo/config.yaml's physics.gravity_y (800 game units/s^2) --
# hardcoded here the same way player_controller.py hardcodes JUMP_IMPULSE, since scripts have no
# API to read engine config.
#
# Facing and animation both track the player: this entity faces whichever side the player is
# currently on (set_facing preserves scale magnitude, only flips its sign -- same convention as
# player_controller.py), and stays in its static "idle" pose except for the brief "attack"
# one-shot animation played only while a shot is actually firing (see level1.yaml's
# "enemy_bomber": idle is frame 0 of the same strip, pinned via fps: 0; attack plays all 5 frames
# once). animation_finished() drives the switch back to idle once the one-shot completes.
#
# Engine API (injected as module global "engine"):
#   engine.emit(name, payload={})
#   engine.get_entity_position(name, axis) -> float -- axis: 'x' or 'y'; 0 if not found
#
# Entity API (injected as module global "self" before each update call):
#   self.get_position(axis) -> float -- axis: 'x' or 'y'
#   self.set_facing(dir_x) -- mirror the sprite to face dir_x's sign
#   self.play_animation(name) -- switch the visible spritesheet ("idle"/"attack")
#   self.reset_animation(name) -- restart a one-shot animation from frame 0
#   self.current_animation() -> str
#   self.animation_finished() -> bool

GRAVITY = 800.0  # game units/s^2, Y-down -- mirrors config.yaml's physics.gravity_y
PLAYER_FEET_OFFSET = 80.0  # player's physics_body offset_y (20) + height (60), see level1.yaml
FIRE_INTERVAL = 3.0  # seconds between shots
FLIGHT_TIME = 1.1  # seconds the bomb takes to reach the player -- controls arc height/speed
FUSE_SECONDS = 1.6  # explodes shortly after landing (FLIGHT_TIME below), not the instant it lands

# Module-level state is safe here (unlike sphere_timer.py) because exactly one GameObject
# ("enemy_bomber") ever uses this script path -- see script/python_engine.go's
# PythonEngine.modules, keyed by script path, not by GameObject.
_cooldown = 1.0  # first shot fires shortly after the scene loads, not instantly


def update(dt):
    global _cooldown

    if self.current_animation() == "attack" and self.animation_finished():
        self.play_animation("idle")

    x0 = self.get_position("x")
    y0 = self.get_position("y")
    x1 = engine.get_entity_position("player", "x")
    y1 = engine.get_entity_position("player", "y")

    self.set_facing(-1 if x1 < x0 else 1)

    _cooldown -= dt
    if _cooldown > 0:
        return
    _cooldown = FIRE_INTERVAL

    y1 += PLAYER_FEET_OFFSET
    vx = (x1 - x0) / FLIGHT_TIME
    vy = (y1 - y0 - 0.5 * GRAVITY * FLIGHT_TIME * FLIGHT_TIME) / FLIGHT_TIME

    self.play_animation("attack")
    self.reset_animation("attack")

    engine.emit("SpawnEntity", {
        "prototype": "sphere_prototype",
        "x": x0,
        "y": y0,
        "vel_x": vx,
        "vel_y": vy,
        "timer_seconds": FUSE_SECONDS,
    })
