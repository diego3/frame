package script

import (
	lua "github.com/yuin/gopher-lua"
)

// goValueToLua converts a Go value to a Lua LValue for use in get_entity().get() and on_event payload.
func goValueToLua(L *lua.LState, v interface{}) lua.LValue {
	if v == nil {
		return lua.LNil
	}
	switch x := v.(type) {
	case string:
		return lua.LString(x)
	case float64:
		return lua.LNumber(x)
	case int:
		return lua.LNumber(x)
	case int64:
		return lua.LNumber(x)
	case bool:
		return lua.LBool(x)
	case map[string]interface{}:
		return mapToLuaTable(L, x)
	default:
		return lua.LNil
	}
}

// mapToLuaTable converts a Go map to a Lua table (for payload in on_event).
func mapToLuaTable(L *lua.LState, m map[string]interface{}) *lua.LTable {
	t := L.NewTable()
	for k, v := range m {
		t.RawSetString(k, goValueToLua(L, v))
	}
	return t
}

// luaValueToGo converts a Lua value to a Go interface{} for use in ScriptEmitted.Payload.
// Supports string, number, bool, and nested tables (map[string]interface{}).
func luaValueToGo(L *lua.LState, v lua.LValue) interface{} {
	switch v.Type() {
	case lua.LTNil:
		return nil
	case lua.LTBool:
		return lua.LVAsBool(v)
	case lua.LTNumber:
		return float64(lua.LVAsNumber(v))
	case lua.LTString:
		return lua.LVAsString(v)
	case lua.LTTable:
		return luaTableToMap(L, v.(*lua.LTable))
	default:
		return nil
	}
}

// luaTableToMap converts a Lua table to map[string]interface{} (string keys only).
// Used for engine.emit(name, payload) so scripts can pass a table as the payload.
func luaTableToMap(L *lua.LState, t *lua.LTable) map[string]interface{} {
	out := make(map[string]interface{})
	t.ForEach(func(k, v lua.LValue) {
		if k.Type() == lua.LTString {
			out[lua.LVAsString(k)] = luaValueToGo(L, v)
		}
	})
	return out
}
