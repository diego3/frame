package script

import (
	"goengine/object"

	lua "github.com/yuin/gopher-lua"
)

// LuaEngine implements Engine using the pure-Go Lua 5.1 VM (gopher-lua).
// All loaded scripts share a single Lua state; the global "self" is rebound
// before each CallScriptUpdate so entity-specific API is correctly scoped.
// This matches the original behaviour described in ADR-005.
type LuaEngine struct {
	vm *VM
}

// NewLuaEngine creates a new LuaEngine backed by a fresh Lua VM.
func NewLuaEngine() *LuaEngine {
	return &LuaEngine{vm: NewVM()}
}

// DoFile loads and evaluates the Lua file at path into the shared VM state.
func (e *LuaEngine) DoFile(path string) error {
	return e.vm.DoFile(path)
}

// DoString loads and evaluates src into the shared VM state. path is unused
// because all Lua scripts share the same VM state; it is present only to
// satisfy the Engine interface.
func (e *LuaEngine) DoString(path, src string) error {
	return e.vm.DoString(src)
}

// RegisterEngineAPI exposes play_sound, switch_scene, quit, emit, and get_entity_position to Lua
// scripts as the global "engine" table (e.g. engine.play_sound("click.wav")).
func (e *LuaEngine) RegisterEngineAPI(
	playSound func(string),
	switchScene func(string),
	quit func(),
	emit func(string, map[string]interface{}),
	getEntityPosition func(string, string) (float64, bool),
) {
	e.vm.RegisterEngine("engine", EngineFuncs(playSound, switchScene, quit, emit, getEntityPosition))
}

// CallScriptUpdate sets the global Lua "self" to the entity API for go_ and
// calls the named function (e.g. "update") with dt as its sole argument.
// The path argument is unused because all Lua scripts share the same VM state;
// it is present only to satisfy the Engine interface.
func (e *LuaEngine) CallScriptUpdate(path, funcName string, go_ *object.GameObject, dt float64) error {
	SetSelf(e.vm.L, go_)
	_, err := e.vm.CallFunc(funcName, lua.LNumber(dt))
	return err
}

// CallOnEvent invokes on_event(name, payload) in the shared Lua state if the
// function is defined. payload is converted to a Lua table.
func (e *LuaEngine) CallOnEvent(name string, payload map[string]interface{}) error {
	if e.vm == nil || e.vm.L == nil {
		return nil
	}
	return CallOnEvent(e.vm.L, name, payload)
}

// Close releases the underlying Lua state. Do not use the engine after Close.
func (e *LuaEngine) Close() {
	if e.vm != nil {
		e.vm.Close()
		e.vm = nil
	}
}

// RawVM returns the underlying Lua VM for direct access when needed.
// Prefer using the Engine interface where possible.
func (e *LuaEngine) RawVM() *VM {
	return e.vm
}
