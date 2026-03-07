package script

import (
	"goengine/object"
	"goengine/physics"

	lua "github.com/yuin/gopher-lua"
)

// SetSelf sets the global Lua "self" to the entity API for the given GameObject.
// Scripts can call self.set_velocity(vx, vy), self.play_animation(name), self.get_intent(key), etc.
// Call this before invoking a script's update(dt) so the script operates on the correct entity.
func SetSelf(L *lua.LState, go_ *object.GameObject) {
	t := L.NewTable()
	// set_velocity(vx, vy) — sets physics body linear velocity
	t.RawSetString("set_velocity", L.NewFunction(func(L *lua.LState) int {
		vx := float64(L.CheckNumber(1))
		vy := float64(L.CheckNumber(2))
		if pb := go_.PhysicsBody(); pb != nil && pb.Body != nil {
			pb.Body.SetLinearVelocity(physics.Vec2{X: vx, Y: vy})
		}
		return 0
	}))
	// play_animation(name)
	t.RawSetString("play_animation", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		if anim := go_.Animator(); anim != nil {
			anim.Play(name)
		}
		return 0
	}))
	// reset_animation(name) — for one-shot anims
	t.RawSetString("reset_animation", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		go_.ResetAnimation(name)
		return 0
	}))
	// current_animation() -> string
	t.RawSetString("current_animation", L.NewFunction(func(L *lua.LState) int {
		if anim := go_.Animator(); anim != nil {
			L.Push(lua.LString(anim.Current))
			return 1
		}
		L.Push(lua.LString(""))
		return 1
	}))
	// animation_finished() -> bool (for current animation if one-shot)
	t.RawSetString("animation_finished", L.NewFunction(func(L *lua.LState) int {
		anim := go_.Animator()
		if anim == nil || anim.Current == "" {
			L.Push(lua.LBool(false))
			return 1
		}
		key := "spritesheet:" + anim.Current
		c := go_.GetComponent(key)
		if c == nil {
			L.Push(lua.LBool(false))
			return 1
		}
		if s, ok := c.(*object.Spritesheet); ok {
			L.Push(lua.LBool(s.Finished()))
			return 1
		}
		L.Push(lua.LBool(false))
		return 1
	}))
	// get_intent(key) -> number or bool; keys: "move_x", "move_y", "dash", "attack", "attack2"
	t.RawSetString("get_intent", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		c := go_.GetComponent("intent_buffer")
		if c == nil {
			pushIntentZero(L, key)
			return 1
		}
		ib, ok := c.(*object.IntentBuffer)
		if !ok {
			pushIntentZero(L, key)
			return 1
		}
		switch key {
		case "move_x":
			L.Push(lua.LNumber(ib.PendingMoveX))
		case "move_y":
			L.Push(lua.LNumber(ib.PendingMoveY))
		case "dash":
			L.Push(lua.LBool(ib.PendingDash))
		case "attack":
			L.Push(lua.LBool(ib.PendingAttack))
		case "attack2":
			L.Push(lua.LBool(ib.PendingAttack2))
		default:
			pushIntentZero(L, key)
		}
		return 1
	}))
	// get_name() -> string
	t.RawSetString("get_name", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(go_.Name))
		return 1
	}))
	L.SetGlobal("self", t)
}

func pushIntentZero(L *lua.LState, key string) {
	switch key {
	case "move_x", "move_y":
		L.Push(lua.LNumber(0))
	default:
		L.Push(lua.LBool(false))
	}
}
