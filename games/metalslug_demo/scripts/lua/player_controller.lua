-- Player controller script (Metal Slug demo, step 3: run + shoot; enhanced with jump).
-- Attach as the "script" component on the "player" GameObject.
--
-- Movement comes from self.get_intent() (IntentBuffer filled by MainMenu on MoveRequested);
-- side-scroller, so only move_x is used. Shooting comes from on_event("AttackRequested") (input
-- action "attack", same convention as games/demo1's dash/attack), which fires engine.emit with a
-- "SpawnProjectile" event; MainMenu (Go) is the one that actually builds the projectile GameObject
-- (see view/scene/mainmenu.go's spawnProjectile) since self has no position getter and the spawn
-- position/size are engine concerns, not script ones.
--
-- Jump is triggered by on_event("JumpRequested") and applies an upward impulse if grounded.
-- Physics gravity (configured in config.yaml) is applied by the engine; script tracks velocity_y
-- across frames and detects ground contact.

local PLAYER_SPEED = 150
local JUMP_FORCE = 400  -- upward impulse in game units/s
local GRAVITY = 800     -- gravity acceleration in game units/s² (must match config.yaml)

-- Facing direction, used as the shot direction; defaults to facing right until the player moves.
local facing_x = 1.0

local pending_shoot = false
local pending_jump = false
local velocity_y = 0
local is_grounded = true
local prev_velocity_y = 0

function on_event(name, _payload)
  if name == "AttackRequested" then
    pending_shoot = true
  elseif name == "JumpRequested" then
    pending_jump = true
  end
end

function update(dt)
  local move_x = self.get_intent("move_x") or 0

  if move_x ~= 0 then
    facing_x = move_x > 0 and 1.0 or -1.0
  end

  -- Apply gravity
  velocity_y = velocity_y + GRAVITY * dt

  -- Ground detection: simple heuristic—if we were falling (prev_velocity_y > 0) and now
  -- velocity is near zero or the physics engine has stopped us, we're grounded.
  -- More reliably: use physics contact (which we'd need to implement), but for now,
  -- assume grounded if velocity_y is very small (clamped by physics engine on impact).
  if velocity_y > 0 then
    is_grounded = true  -- falling and likely to contact ground soon
  elseif velocity_y < -50 then
    is_grounded = false  -- clearly in the air (upward)
  end
  -- else: velocity_y near zero, keep previous grounded state

  -- Apply jump if requested and grounded
  if pending_jump and is_grounded then
    velocity_y = -JUMP_FORCE
    is_grounded = false
    pending_jump = false
  end

  self.set_velocity(move_x * PLAYER_SPEED, velocity_y)

  if pending_shoot then
    engine.emit("SpawnProjectile", { dir_x = facing_x, dir_y = 0 })
    pending_shoot = false
  end

  prev_velocity_y = velocity_y
end
