// Package script provides script execution backends for game logic.
// Two backends are available: LuaEngine (gopher-lua, pure Go Lua 5.1) and PythonEngine (gpython, pure Go Python 3.4).
// Use NewEngine to create the right backend based on a configuration string.
package script

import "goengine/object"

// Engine is the interface for running game scripts from Go.
// Both Lua and Python backends implement this interface, allowing the game engine
// to be flexible about which scripting language is used.
//
// Typical lifecycle:
//  1. Create with NewEngine(engineType)
//  2. Call RegisterEngineAPI once to expose engine callbacks to scripts
//  3. Call DoFile for each script file to load (idempotency is the caller's responsibility)
//  4. Each frame, call CallScriptUpdate for entities that have a script component
//  5. On script events (e.g. DashRequested), call CallOnEvent
//  6. Call Close when the engine is no longer needed (e.g. scene unload)
type Engine interface {
	// DoFile loads and executes the script at path, making its functions available
	// for subsequent CallScriptUpdate and CallOnEvent calls.
	DoFile(path string) error

	// DoString loads and executes src as a script, using path as its identifying
	// name for later CallScriptUpdate lookups. Used when scripts come from an
	// embedded fs.FS (e.g. WASM builds) instead of the OS filesystem, where only
	// the source bytes and a logical path are available.
	DoString(path, src string) error

	// RegisterEngineAPI registers Go callbacks so scripts can call engine.play_sound,
	// engine.switch_scene, engine.quit, engine.emit, and engine.get_entity_position.
	RegisterEngineAPI(
		playSound func(path string),
		switchScene func(sceneID string),
		quit func(),
		emit func(name string, payload map[string]interface{}),
		getEntityPosition func(name, axis string) (float64, bool),
	)

	// CallScriptUpdate sets the entity context (self) for go_ and calls the named
	// update function (typically "update") in the script identified by path, passing dt.
	// Returns nil if the function is not defined (not an error).
	CallScriptUpdate(path, funcName string, go_ *object.GameObject, dt float64) error

	// CallOnEvent invokes on_event(name, payload) across all loaded scripts that define it.
	// Returns nil if on_event is not defined.
	CallOnEvent(name string, payload map[string]interface{}) error

	// Close releases all resources held by the engine. Do not use the engine after Close.
	Close()
}

// NewEngine returns a script engine for the given type string.
// Supported values: "lua" (default, pure-Go Lua 5.1 via gopher-lua)
// and "python" (pure-Go Python 3.4 via gpython).
// An empty or unrecognised value defaults to Lua.
func NewEngine(engineType string) Engine {
	switch engineType {
	case "python":
		return NewPythonEngine()
	default:
		return NewLuaEngine()
	}
}
