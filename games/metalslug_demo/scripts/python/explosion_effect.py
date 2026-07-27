# Explosion VFX script (Python) — Metal Slug demo. Attach as the "script" component on an
# "explosion_prototype" clone (see level1.yaml and sphere_timer.py, which spawns one of these via
# engine.emit("SpawnEntity", ...) at the moment a sphere's fuse runs out).
#
# The entity's only visual is a one-shot spritesheet (loop: false); this script's whole job is to
# notice when that animation is done and remove the entity. animation_finished() is the primary
# signal. self.get_timer() is a defensive fallback (the prototype's "timer" component, sized
# comfortably longer than the animation) in case FPS/frame-count are ever misconfigured in YAML
# and the animation never reports finished -- without it a bad config would leak an entity that
# never draws anything but also never goes away.
#
# Entity API (injected as module global "self" before each update call):
#   self.animation_finished() -> bool
#   self.get_timer() -> float
#   self.set_timer(seconds)
#   self.destroy() -- deactivates the GameObject (removed from Update/Draw)


def update(dt):
    remaining = self.get_timer() - dt
    self.set_timer(remaining)

    if self.animation_finished() or remaining <= 0:
        self.destroy()
