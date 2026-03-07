-- Knight controller script. Attach as script component to the "knight" GameObject.
-- Movement comes from self.get_intent (IntentBuffer filled by MainMenu on MoveRequested).
-- Dash/attack come from on_event("DashRequested"), etc., for decoupling from scene logic.

local KNIGHT_SPEED = 120
local DASH_IMPULSE = 280
local DASH_COOLDOWN_SEC = 0.5
local DASH_DURATION_SEC = 0.12
local DASH_DIR_WHEN_IDLE = 1.0

local dash_cooldown = 0
local dash_active_until = 0
local last_move_x = 0
local last_move_y = 0

-- One-shot intents set by on_event; cleared after read in update()
local pending_dash = false
local pending_attack = false
local pending_attack2 = false

function on_event(name, _payload)
  if name == "DashRequested" then
    pending_dash = true
  elseif name == "AttackRequested" then
    pending_attack = true
  elseif name == "Attack2Requested" then
    pending_attack2 = true
  end
end

local function read_intents()
  return {
    move_x = self.get_intent("move_x") or 0,
    move_y = self.get_intent("move_y") or 0,
    dash = pending_dash,
    attack = pending_attack,
    attack2 = pending_attack2,
  }
end

local function clear_one_shot_intents()
  pending_dash = false
  pending_attack = false
  pending_attack2 = false
end

local function tick_cooldowns(dt)
  if dash_cooldown > 0 then
    dash_cooldown = dash_cooldown - dt
  end

  if dash_active_until > 0 then
    dash_active_until = dash_active_until - dt
  end
end

local function run_velocity(move_x, move_y)
  return move_x * KNIGHT_SPEED, move_y * KNIGHT_SPEED
end

local function dash_impulse_vector(vx, vy)
  if vx ~= 0 or vy ~= 0 then
    local len = math.sqrt(vx * vx + vy * vy)
    if len > 0 then
      len = 1.0 / len
      return vx * len * DASH_IMPULSE, vy * len * DASH_IMPULSE
    end
  end
  local dx = last_move_x * DASH_IMPULSE
  local dy = last_move_y * DASH_IMPULSE
  if dx == 0 and dy == 0 then
    dx = DASH_DIR_WHEN_IDLE * DASH_IMPULSE
  end
  return dx, dy
end

local function try_dash(intents, vx, vy)
  if not intents.dash or dash_cooldown > 0 then return false end
  local dx, dy = dash_impulse_vector(vx, vy)
  if dx == 0 and dy == 0 then return false end
  self.set_velocity(dx, dy)
  dash_cooldown = DASH_COOLDOWN_SEC
  dash_active_until = DASH_DURATION_SEC
  self.play_animation("dash")
  self.reset_animation("dash")
  return true
end

local function is_one_shot_anim(name)
  return name == "attack" or name == "attack2" or name == "dash"
end

local function pick_movement_animation(intents)
  local cur = self.current_animation()
  if is_one_shot_anim(cur) then
    if self.animation_finished() then
      self.play_animation("idle")
    end
  elseif intents.move_x ~= 0 or intents.move_y ~= 0 then
    self.play_animation("run")
  else
    self.play_animation("idle")
  end
end

function update(dt)
  local intents = read_intents()
  clear_one_shot_intents()
  tick_cooldowns(dt)

  if intents.move_x ~= 0 or intents.move_y ~= 0 then
    last_move_x = intents.move_x
    last_move_y = intents.move_y
  end

  local vx, vy = run_velocity(intents.move_x, intents.move_y)
  local did_dash = try_dash(intents, vx, vy)

  if not did_dash and dash_active_until <= 0 then
    self.set_velocity(vx, vy)
  end

  if intents.attack then
    self.play_animation("attack")
    self.reset_animation("attack")
  elseif intents.attack2 then
    self.play_animation("attack2")
    self.reset_animation("attack2")
  end

  pick_movement_animation(intents)
end
