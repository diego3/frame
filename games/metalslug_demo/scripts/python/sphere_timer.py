# Sphere timer script (Python) — Metal Slug demo "process manager" stress test.
# Attach as the "script" component on a sphere cloned from "sphere_prototype" (see level1.yaml
# and view/scene/mainmenu.go's updateSphereSpawner, which spawns these at a random position above
# the level with a randomized fuse and lets Box2D gravity make them fall).
#
# All countdown state lives in the entity's Timer component (self.get_timer()/set_timer()), not in
# a module-level variable here: every GameObject that uses this script path shares this module's
# globals (see script/python_engine.go's PythonEngine.modules, keyed by script path), so with
# several spheres falling at once a plain global would mix up their countdowns. Reading the
# "before" value from the component on every call, instead of caching it in a global, is what
# keeps this script correct with any number of concurrent spheres -- and correspondingly easy to
# unit test one instance at a time (see sphere_timer_test.py).
#
# Engine API (injected as module global "engine"):
#   engine.emit(name, payload={}) -- used to spawn an "explosion_prototype" VFX clone (see
#   explosion_effect.py) at this sphere's position the moment it explodes, via the same generic
#   Prototype-clone mechanism game_manager.py uses to spawn spheres in the first place (see
#   MainMenu.spawnEntity). Deciding that an explosion happens, and where, is this script's rule to
#   make -- MainMenu only knows how to clone a named prototype, not why. The same moment also
#   emits "ShakeCamera" (WorldScene.shakeCamera) -- since this script is the one place that decides
#   "a sphere just exploded", regardless of whether it got here as the falling-sphere hazard
#   (game_manager.py) or an enemy_bomber.py bomb, both trigger the shake for free.
#
# Entity API (injected as module global "self" before each update call):
#   self.get_timer() -> float
#   self.set_timer(seconds)
#   self.get_name() -> str
#   self.get_position(axis) -> float -- axis: 'x' or 'y'
#   self.destroy() -- deactivates the GameObject (removed from Update/Draw)

import math

SHAKE_DURATION = 0.3  # seconds the camera shake lasts on explosion
SHAKE_MAGNITUDE = 8  # peak per-axis camera offset in world units


def update(dt):
    remaining_before = self.get_timer()
    remaining_after = remaining_before - dt
    self.set_timer(remaining_after)

    second_before = int(math.ceil(remaining_before)) if remaining_before > 0 else 0
    second_after = int(math.ceil(remaining_after)) if remaining_after > 0 else 0
    if 0 < second_after < second_before:
        print(self.get_name() + ": " + str(second_after))

    if remaining_after <= 0:
        print(self.get_name() + ": exploded")
        engine.emit("SpawnEntity", {
            "prototype": "explosion_prototype",
            "name": self.get_name() + "_explosion",
            "x": self.get_position("x"),
            "y": self.get_position("y"),
        })
        engine.emit("ShakeCamera", {"duration": SHAKE_DURATION, "magnitude": SHAKE_MAGNITUDE})
        self.destroy()
