-- Enemy walk script (Metal Slug demo, step 5). Attach as the "script" component on an enemy
-- GameObject (alongside a physics_body so set_velocity actually moves it, same as the player).
--
-- Simplest possible behavior, per the build plan: walks left at a constant velocity, no
-- pathfinding, no state machine. HP and death (on projectile contact) are handled in Go — see
-- view/scene/mainmenu.go's updateHitDetection — since self has no position getter and hit
-- detection needs to compare against every live projectile each frame, which is an engine concern,
-- not something a single entity's script can do on its own.
--
-- Named enemy_update, not update: LuaEngine runs every loaded script in one shared global VM
-- state (see script/lua_engine.go) — a plain "update" here would silently overwrite
-- player_controller.lua's own "update" global the moment this script loads, since Lua functions
-- declared at chunk level are global by default. The scene's script component sets
-- update_func: "enemy_update" to call this one by its distinct name instead. (PythonEngine doesn't
-- need this — each script gets its own isolated module — so player_controller.py/enemy_walk.py
-- both keep the plain "update" name.)

local ENEMY_SPEED = 60

function enemy_update(dt)
  self.set_velocity(-ENEMY_SPEED, 0)
end
