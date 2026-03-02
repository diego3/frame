package script

import (
	lua "github.com/yuin/gopher-lua"
)

// EngineFuncs returns a map of Lua-callable engine functions. Pass your Go callbacks so scripts
// can call engine.play_sound(path), engine.switch_scene(scene_id), and engine.quit().
// The returned map is suitable for VM.RegisterEngine("engine", EngineFuncs(...)).
func EngineFuncs(
	playSound func(path string),
	switchScene func(sceneID string),
	quit func(),
) map[string]func(L *lua.LState) int {
	return map[string]func(L *lua.LState) int{
		"play_sound": func(L *lua.LState) int {
			path := L.CheckString(1)
			playSound(path)
			return 0
		},
		"switch_scene": func(L *lua.LState) int {
			sceneID := L.CheckString(1)
			switchScene(sceneID)
			return 0
		},
		"quit": func(L *lua.LState) int {
			quit()
			return 0
		},
	}
}
