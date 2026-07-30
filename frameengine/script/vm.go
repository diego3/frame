// Package script provides a pure-Go Lua 5.1 VM for running game scripts.
// Scripts can call into the engine via registered functions (e.g. engine.play_sound, engine.switch_scene).
// See ADR-005 (Option 1: Pure Go VM).
package script

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// VM wraps a Lua state. Create with NewVM, register engine functions, then run DoFile or DoString.
// Call Close when done (e.g. on scene unload) to release the state.
type VM struct {
	L *lua.LState
}

// NewVM returns a new Lua VM. Call RegisterEngine to expose Go functions to scripts, then DoFile/DoString.
func NewVM() *VM {
	return &VM{L: lua.NewState()}
}

// RegisterEngine creates a global Lua table named `name` (e.g. "engine") and registers each
// function in fns. From Lua, scripts call e.g. engine.play_sound("footstep.ogg").
// Each Go function receives the LState and returns the number of return values (0 for none).
func (v *VM) RegisterEngine(name string, fns map[string]func(L *lua.LState) int) {
	t := v.L.NewTable()
	for k, fn := range fns {
		t.RawSetString(k, v.L.NewFunction(fn))
	}
	v.L.SetGlobal(name, t)
}

// DoFile runs the Lua file at path. Returns an error if the file cannot be read or the script errors.
func (v *VM) DoFile(path string) error {
	if err := v.L.DoFile(path); err != nil {
		return fmt.Errorf("script dofile %q: %w", path, err)
	}
	return nil
}

// DoString runs the Lua snippet. Returns an error if the script errors.
func (v *VM) DoString(s string) error {
	if err := v.L.DoString(s); err != nil {
		return fmt.Errorf("script dostring: %w", err)
	}
	return nil
}

// CallFunc calls the global Lua function named name with the given arguments.
// Returns the function's return values, or an error if the function is missing or fails.
// Use from Go when you need to invoke a script function (e.g. on_trigger).
func (v *VM) CallFunc(name string, args ...lua.LValue) ([]lua.LValue, error) {
	fn := v.L.GetGlobal(name)
	if fn.Type() != lua.LTFunction {
		return nil, fmt.Errorf("script: global %q is not a function (got %s)", name, fn.Type().String())
	}
	v.L.Push(fn)
	for _, a := range args {
		v.L.Push(a)
	}
	if err := v.L.PCall(len(args), lua.MultRet, nil); err != nil {
		return nil, fmt.Errorf("script call %q: %w", name, err)
	}
	n := v.L.GetTop()
	var ret []lua.LValue
	for i := 1; i <= n; i++ {
		ret = append(ret, v.L.Get(i))
	}
	v.L.Pop(n)
	return ret, nil
}

// Close releases the Lua state. Call when the VM is no longer needed (e.g. scene unload).
// Do not use the VM after Close.
func (v *VM) Close() {
	if v.L != nil {
		v.L.Close()
		v.L = nil
	}
}

// CallOnEvent invokes the global Lua function on_event(name, payload) if it exists.
// Use when delivering ScriptEmitted to the VM so scripts can react via function on_event(name, payload).
// Payload is converted to a Lua table. If on_event is not defined or not a function, does nothing.
func CallOnEvent(L *lua.LState, name string, payload map[string]interface{}) error {
	fn := L.GetGlobal("on_event")
	if fn.Type() != lua.LTFunction {
		return nil
	}
	L.Push(fn)
	L.Push(lua.LString(name))
	L.Push(mapToLuaTable(L, payload))
	if err := L.PCall(2, 0, nil); err != nil {
		return fmt.Errorf("on_event: %w", err)
	}
	return nil
}
