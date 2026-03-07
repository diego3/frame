package object

// Script attaches a Lua script to a GameObject. The script is run each frame by the scene's
// script runner (shared VM per scene). Script is data-only; it does not implement Updater.
// The runner sets the global "self" to the entity API for this object and calls the UpdateFunc.
type Script struct {
	Path         string // path to the Lua file (e.g. "scripts/knight_controller.lua")
	UpdateFuncName string // name of the global function to call each frame (default "update")
}

// Type implements Component.
func (Script) Type() string { return "script" }
