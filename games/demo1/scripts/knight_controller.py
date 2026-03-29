# Knight controller script (Python version).
# Equivalent to knight_controller.lua — attach as the "script" component on the "knight" GameObject.
#
# Movement comes from self.get_intent() (IntentBuffer filled by MainMenu on MoveRequested).
# Dash/attack are delivered via on_event("DashRequested") etc. from input script_events config.
#
# Engine API (injected as module global "engine"):
#   engine.play_sound(path)
#   engine.switch_scene(scene_id)
#   engine.quit()
#   engine.emit(name, payload={})
#
# Entity API (injected as module global "self" before each update call):
#   self.set_velocity(vx, vy)
#   self.play_animation(name)
#   self.reset_animation(name)
#   self.current_animation()  -> str
#   self.animation_finished() -> bool
#   self.get_intent(key)      -> float  (keys: "move_x", "move_y")
#   self.get_name()           -> str

KNIGHT_SPEED = 120
DASH_IMPULSE = 280
DASH_COOLDOWN_SEC = 0.5
DASH_DURATION_SEC = 0.12
DASH_DIR_WHEN_IDLE = 1.0

dash_cooldown = 0.0
dash_active_until = 0.0
last_move_x = 0.0
last_move_y = 0.0

# One-shot intents set by on_event; cleared after consumed in update()
pending_dash = False
pending_attack = False
pending_attack2 = False


def on_event(name, _payload):
    global pending_dash, pending_attack, pending_attack2
    if name == "DashRequested":
        pending_dash = True
    elif name == "AttackRequested":
        pending_attack = True
    elif name == "Attack2Requested":
        pending_attack2 = True


def _read_intents():
    return {
        "move_x": self.get_intent("move_x"),
        "move_y": self.get_intent("move_y"),
        "dash": pending_dash,
        "attack": pending_attack,
        "attack2": pending_attack2,
    }


def _clear_one_shot_intents():
    global pending_dash, pending_attack, pending_attack2
    pending_dash = False
    pending_attack = False
    pending_attack2 = False


def _tick_cooldowns(dt):
    global dash_cooldown, dash_active_until
    if dash_cooldown > 0:
        dash_cooldown = dash_cooldown - dt
    if dash_active_until > 0:
        dash_active_until = dash_active_until - dt


def _run_velocity(move_x, move_y):
    return move_x * KNIGHT_SPEED, move_y * KNIGHT_SPEED


def _dash_impulse_vector(vx, vy):
    if vx != 0 or vy != 0:
        length = (vx * vx + vy * vy) ** 0.5
        if length > 0:
            inv = 1.0 / length
            return vx * inv * DASH_IMPULSE, vy * inv * DASH_IMPULSE
    dx = last_move_x * DASH_IMPULSE
    dy = last_move_y * DASH_IMPULSE
    if dx == 0 and dy == 0:
        dx = DASH_DIR_WHEN_IDLE * DASH_IMPULSE
    return dx, dy


def _try_dash(intents, vx, vy):
    global dash_cooldown, dash_active_until
    if not intents["dash"] or dash_cooldown > 0:
        return False
    dx, dy = _dash_impulse_vector(vx, vy)
    if dx == 0 and dy == 0:
        return False
    self.set_velocity(dx, dy)
    dash_cooldown = DASH_COOLDOWN_SEC
    dash_active_until = DASH_DURATION_SEC
    self.play_animation("dash")
    return True


def _is_one_shot_anim(name):
    return name == "attack" or name == "attack2" or name == "dash"


def _pick_movement_animation(intents):
    if intents["move_x"] != 0 or intents["move_y"] != 0:
        self.play_animation("run")
    else:
        self.play_animation("idle")


# Full update — called each frame by the engine with dt in seconds.
# One-shot animations (attack, attack2, dash) are reset when they finish, not when started.
def update(dt):
    global last_move_x, last_move_y

    intents = _read_intents()
    _tick_cooldowns(dt)

    if intents["move_x"] != 0 or intents["move_y"] != 0:
        last_move_x = intents["move_x"]
        last_move_y = intents["move_y"]

    vx, vy = _run_velocity(intents["move_x"], intents["move_y"])
    did_dash = _try_dash(intents, vx, vy)

    if not did_dash and dash_active_until <= 0:
        self.set_velocity(vx, vy)

    # Start one-shot animations (reset happens when they finish, not here)
    if pending_attack:
        self.play_animation("attack")
    elif pending_attack2:
        self.play_animation("attack2")

    # When a one-shot finishes: transition to idle, reset the animation, clear intents
    cur = self.current_animation()
    if _is_one_shot_anim(cur) and self.animation_finished():
        self.play_animation("idle")
        self.reset_animation(cur)
        _clear_one_shot_intents()

    # When not in a one-shot, choose idle or run based on movement
    if not _is_one_shot_anim(self.current_animation()):
        _pick_movement_animation(intents)
