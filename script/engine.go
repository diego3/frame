package script

import (
	lua "github.com/yuin/gopher-lua"
)

// EngineFuncs returns a map of Lua-callable engine functions. Pass your Go callbacks so scripts
// can call engine.play_sound, engine.switch_scene, engine.quit, and engine.emit(name, payload).
// The returned map is suitable for VM.RegisterEngine("engine", EngineFuncs(...)).
func EngineFuncs(
	playSound func(path string),
	switchScene func(sceneID string),
	quit func(),
	emit func(name string, payload map[string]interface{}),
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
		"emit": func(L *lua.LState) int {
			name := L.CheckString(1)
			var payload map[string]interface{}
			if L.GetTop() >= 2 && L.Get(2).Type() == lua.LTTable {
				payload = luaTableToMap(L, L.CheckTable(2))
			}
			if payload == nil {
				payload = make(map[string]interface{})
			}
			emit(name, payload)
			return 0
		},
	}
}
