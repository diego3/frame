-- Sample script callable from the engine (Option 1: Pure Go Lua VM, ADR-005).
-- From Go: game.ScriptVM():DoFile("scripts/sample.lua") or DoString(...).
-- Engine API:
--   engine.play_sound(path)   -- play a WAV (path must be loaded)
--   engine.switch_scene(id)   -- request scene change
--   engine.quit()             -- request application exit

-- Example: call engine from Lua
-- engine.play_sound("assets/click.wav")
-- engine.switch_scene("main_menu")
-- engine.quit()

-- Define a function that Go can call via VM.CallFunc("on_trigger", ...)
function on_trigger(obj_name)
  -- obj_name is a string passed from Go
  return "triggered:" .. obj_name
end
