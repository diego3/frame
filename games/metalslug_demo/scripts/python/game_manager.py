# Game manager script (Python) — Metal Slug demo domain rule: periodically drop a bomb-like
# sphere from the sky at a random position with a randomized fuse. Attach as the "script"
# component on the invisible "game_manager" GameObject (transform only, no visual/physics — see
# level1.yaml). Deciding *when* to spawn, *how long* the fuse is, and *where* it falls from are
# game rules specific to this demo, not generic engine behavior, so they live here rather than in
# view/scene/mainmenu.go (a reusable scene type shared by other games/scenes). The engine only
# provides the generic mechanism -- cloning a named prototype and placing it in the world -- via
# engine.emit("SpawnEntity", {...}), the same Prototype-pattern plumbing spawnProjectile already
# uses for "SpawnProjectile" (see MainMenu.spawnEntity).
#
# gpython (this engine's Python backend, pure-Go Python 3.4) has no "random" module in its stdlib
# subset, so a small xorshift32 PRNG is implemented below, seeded from time.time() (time.time() is
# implemented in gpython's stdlib; time.perf_counter(), the usual choice, raises
# NotImplementedError there -- confirmed directly against gpython v0.2.0).
#
# Engine API used (injected as module global "engine"):
#   engine.emit(name, payload={})
#
# Entity API used: none -- game_manager has no position/velocity/etc. of its own.

import time

SPAWN_INTERVAL_MIN = 1.5  # seconds between spawns, low end
SPAWN_INTERVAL_MAX = 3.5  # seconds between spawns, high end
FUSE_MIN = 3.0  # seconds before a spawned sphere explodes, low end
FUSE_MAX = 6.0  # seconds before a spawned sphere explodes, high end
SPAWN_Y = -40.0  # above the level's top edge; falls into view once gravity takes over
LEVEL_WIDTH = 2400.0  # mirrors layout.level_width in config.yaml

_rng_state = int(time.time() * 1000000) & 0xffffffff

# Counts down to the next spawn; module-level state is safe here (unlike sphere_timer.py) because
# exactly one GameObject ("game_manager") ever uses this script path -- see
# script/python_engine.go's PythonEngine.modules, keyed by script path.
_next_spawn_in = 1.0  # first sphere drops shortly after the scene loads
_spawn_count = 0


def _random():
    """xorshift32 in [0, 1). No external seed source needed beyond module load time, since this
    only has to look random to a player, not withstand statistical scrutiny."""
    global _rng_state
    x = _rng_state
    x ^= (x << 13) & 0xffffffff
    x ^= (x >> 17)
    x ^= (x << 5) & 0xffffffff
    _rng_state = x & 0xffffffff
    return _rng_state / 4294967296.0


def _random_range(lo, hi):
    return lo + _random() * (hi - lo)


def update(dt):
    global _next_spawn_in, _spawn_count

    _next_spawn_in -= dt
    if _next_spawn_in > 0:
        return
    _next_spawn_in = _random_range(SPAWN_INTERVAL_MIN, SPAWN_INTERVAL_MAX)

    _spawn_count += 1
    engine.emit("SpawnEntity", {
        "prototype": "sphere_prototype",
        "name": "sphere_" + str(_spawn_count),
        "x": _random_range(0, LEVEL_WIDTH),
        "y": SPAWN_Y,
        "timer_seconds": _random_range(FUSE_MIN, FUSE_MAX),
    })
