# Enemy walk script (Python version) — Metal Slug demo, step 5.
# Equivalent to enemy_walk.lua — attach as the "script" component on an enemy GameObject
# (alongside a physics_body so set_velocity actually moves it, same as the player).
#
# Simplest possible behavior, per the build plan: walks left at a constant velocity, no
# pathfinding, no state machine. HP and death (on projectile contact) are handled in Go — see
# view/scene/mainmenu.go's updateHitDetection — since self has no position getter and hit
# detection needs to compare against every live projectile each frame, which is an engine concern,
# not something a single entity's script can do on its own.
#
# Entity API (injected as module global "self" before each update call):
#   self.set_velocity(vx, vy)

ENEMY_SPEED = 60


def update(dt):
    self.set_velocity(-ENEMY_SPEED, 0)
