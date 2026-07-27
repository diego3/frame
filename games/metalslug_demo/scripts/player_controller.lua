-- Player controller script (Metal Slug demo, step 3: run + shoot).
-- Attach as the "script" component on the "player" GameObject.
--
-- Movement comes from self.get_intent() (IntentBuffer filled by MainMenu on MoveRequested);
-- side-scroller, so only move_x is used. Shooting comes from on_event("AttackRequested") (input
-- action "attack", same convention as games/demo1's dash/attack), which fires engine.emit with a
-- "SpawnProjectile" event; MainMenu (Go) is the one that actually builds the projectile GameObject
-- (see view/scene/mainmenu.go's spawnProjectile) since self has no position getter and the spawn
-- position/size are engine concerns, not script ones.
--
-- Projectile movement/off-screen deactivation isn't implemented yet (build order step 4) — a
-- spawned projectile is visible but stationary until that lands.

local PLAYER_SPEED = 150

-- Facing direction, used as the shot direction; defaults to facing right until the player moves.
local facing_x = 1.0

local pending_shoot = false

function on_event(name, _payload)
  if name == "AttackRequested" then
    pending_shoot = true
  end
end

function update(dt)
  local move_x = self.get_intent("move_x") or 0

  if move_x ~= 0 then
    facing_x = move_x > 0 and 1.0 or -1.0
  end

  self.set_velocity(move_x * PLAYER_SPEED, 0)

  if pending_shoot then
    engine.emit("SpawnProjectile", { dir_x = facing_x, dir_y = 0 })
    pending_shoot = false
  end
end
