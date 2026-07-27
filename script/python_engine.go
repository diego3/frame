package script

import (
	"fmt"
	"goengine/object"
	"goengine/physics"

	"github.com/go-python/gpython/py"
	// stdlib initialises the gpython runtime and registers built-in modules
	// (math, time, sys, etc.). Must be imported for py.NewContext to work.
	_ "github.com/go-python/gpython/stdlib"
)

// PythonEngine implements Engine using gpython — a pure-Go Python 3.4 interpreter.
// Each script file is loaded into its own py.Module so scripts do not share global
// state and cannot accidentally clobber each other's variables.
//
// The "engine" namespace (engine.play_sound, engine.switch_scene, engine.quit,
// engine.emit) and the entity "self" (self.set_velocity, self.play_animation, …)
// are injected directly into each module's globals before the script runs /
// before each update call, mirroring how LuaEngine exposes the same API via
// global Lua tables.
type PythonEngine struct {
	ctx     py.Context
	modules map[string]*py.Module // script path → loaded module

	// engine API callbacks registered via RegisterEngineAPI
	playSound   func(string)
	switchScene func(string)
	quit        func()
	emit        func(string, map[string]interface{})
}

// NewPythonEngine creates a new PythonEngine with a fresh gpython context.
func NewPythonEngine() *PythonEngine {
	return &PythonEngine{
		ctx:     py.NewContext(py.DefaultContextOpts()),
		modules: make(map[string]*py.Module),
	}
}

// RegisterEngineAPI stores the Go callbacks so they can be injected into every
// script module that is subsequently loaded via DoFile.
func (e *PythonEngine) RegisterEngineAPI(
	playSound func(string),
	switchScene func(string),
	quit func(),
	emit func(string, map[string]interface{}),
) {
	e.playSound = playSound
	e.switchScene = switchScene
	e.quit = quit
	e.emit = emit
}

// newScriptModule creates a fresh module for a script and injects the engine
// API into its globals so the script can call engine.play_sound(...) etc.
func (e *PythonEngine) newScriptModule() (*py.Module, error) {
	impl := &py.ModuleImpl{
		Info: py.ModuleInfo{Name: "__main__"},
	}
	module, err := e.ctx.ModuleInit(impl)
	if err != nil {
		return nil, err
	}
	module.Globals["engine"] = e.buildEngineObject()
	return module, nil
}

// DoFile loads and executes the Python file at path.
// The script runs inside a dedicated py.Module; the "engine" object is injected
// into that module's globals before execution so scripts can call engine.*
// functions at module level if they wish.
// The loaded module is retained so CallScriptUpdate can invoke "update(dt)" later.
func (e *PythonEngine) DoFile(path string) error {
	module, err := e.newScriptModule()
	if err != nil {
		return fmt.Errorf("python: module init for %q: %w", path, err)
	}

	if _, err = py.RunFile(e.ctx, path, py.CompileOpts{}, module); err != nil {
		return fmt.Errorf("python dofile %q: %w", path, err)
	}

	e.modules[path] = module
	return nil
}

// DoString compiles and executes the Python source src, identified by path for
// later CallScriptUpdate lookups. Used when scripts come from an embedded
// fs.FS (e.g. WASM builds) instead of the OS filesystem, where only the
// source bytes and a logical path are available. Mirrors DoFile's compile
// mode (py.ExecMode, matching how gpython itself compiles .py files).
func (e *PythonEngine) DoString(path, src string) error {
	module, err := e.newScriptModule()
	if err != nil {
		return fmt.Errorf("python: module init for %q: %w", path, err)
	}

	code, err := py.Compile(src, path, py.ExecMode, 0, true)
	if err != nil {
		return fmt.Errorf("python compile %q: %w", path, err)
	}
	if _, err = py.RunCode(e.ctx, code, path, module); err != nil {
		return fmt.Errorf("python dostring %q: %w", path, err)
	}

	e.modules[path] = module
	return nil
}

// CallScriptUpdate injects the entity API as the "self" global into the module
// for path, then calls funcName(dt). If funcName is not defined the call is a
// no-op (not an error), matching the Lua engine behaviour.
func (e *PythonEngine) CallScriptUpdate(path, funcName string, go_ *object.GameObject, dt float64) error {
	module, ok := e.modules[path]
	if !ok {
		return fmt.Errorf("python: script %q not loaded; call DoFile first", path)
	}

	// Rebind "self" to the current entity before every update call
	module.Globals["self"] = e.buildSelfObject(go_)

	fn, ok := module.Globals[funcName]
	if !ok {
		return nil // function not defined — not an error
	}

	if _, err := py.Call(fn, py.Tuple{py.Float(dt)}, nil); err != nil {
		return fmt.Errorf("python call %q in %q: %w", funcName, path, err)
	}
	return nil
}

// CallOnEvent calls on_event(name, payload) in every loaded script module that
// defines it. Payload is converted to a Python dict (StringDict) before passing.
func (e *PythonEngine) CallOnEvent(name string, payload map[string]interface{}) error {
	for path, module := range e.modules {
		fn, ok := module.Globals["on_event"]
		if !ok {
			continue
		}
		pyPayload := goMapToPyDict(payload)
		if _, err := py.Call(fn, py.Tuple{py.String(name), pyPayload}, nil); err != nil {
			return fmt.Errorf("python on_event in %q: %w", path, err)
		}
	}
	return nil
}

// Close releases the gpython context and clears the module map.
// Do not use the engine after Close.
func (e *PythonEngine) Close() {
	if e.ctx != nil {
		e.ctx.Close()
		e.ctx = nil
	}
	e.modules = nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildEngineObject constructs a py.Module whose globals contain the four
// engine API methods (play_sound, switch_scene, quit, emit) as callable
// py.Method objects.  Python scripts access them as engine.play_sound(…), etc.
func (e *PythonEngine) buildEngineObject() *py.Module {
	m := &py.Module{
		ModuleImpl: &py.ModuleImpl{Info: py.ModuleInfo{Name: "engine"}},
		Globals:    py.StringDict{},
	}

	m.Globals["play_sound"] = py.MustNewMethod("play_sound",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "play_sound() requires 1 argument")
			}
			path, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			if e.playSound != nil {
				e.playSound(path)
			}
			return py.None, nil
		}, 0, "play_sound(path) -- play a sound asset")

	m.Globals["switch_scene"] = py.MustNewMethod("switch_scene",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "switch_scene() requires 1 argument")
			}
			id, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			if e.switchScene != nil {
				e.switchScene(id)
			}
			return py.None, nil
		}, 0, "switch_scene(scene_id) -- request a scene transition")

	m.Globals["quit"] = py.MustNewMethod("quit",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if e.quit != nil {
				e.quit()
			}
			return py.None, nil
		}, 0, "quit() -- request application exit")

	m.Globals["emit"] = py.MustNewMethod("emit",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "emit() requires at least 1 argument")
			}
			name, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			payload := make(map[string]interface{})
			if len(args) >= 2 {
				if d, ok := args[1].(py.StringDict); ok {
					payload = pyDictToGoMap(d)
				}
			}
			if e.emit != nil {
				e.emit(name, payload)
			}
			return py.None, nil
		}, 0, "emit(name[, payload]) -- emit a ScriptEmitted event")

	return m
}

// buildSelfObject constructs a per-call py.Module whose globals expose the
// entity API (set_velocity, play_animation, …) for the given GameObject.
// Methods capture go_ via closure so the Python "self" variable always
// refers to the correct entity for the current update frame.
func (e *PythonEngine) buildSelfObject(go_ *object.GameObject) *py.Module {
	m := &py.Module{
		ModuleImpl: &py.ModuleImpl{Info: py.ModuleInfo{Name: "self"}},
		Globals:    py.StringDict{},
	}

	m.Globals["set_velocity"] = py.MustNewMethod("set_velocity",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 2 {
				return nil, py.ExceptionNewf(py.TypeError, "set_velocity() requires 2 arguments")
			}
			vx, err := pyToFloat64(args[0])
			if err != nil {
				return nil, err
			}
			vy, err := pyToFloat64(args[1])
			if err != nil {
				return nil, err
			}
			if pb := go_.PhysicsBody(); pb != nil && pb.Body != nil {
				pb.Body.SetLinearVelocity(physics.Vec2{X: vx, Y: vy})
			}
			return py.None, nil
		}, 0, "set_velocity(vx, vy) -- set the physics body linear velocity")

	m.Globals["get_velocity"] = py.MustNewMethod("get_velocity",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "get_velocity() requires 1 argument")
			}
			axis, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			pb := go_.PhysicsBody()
			if pb == nil || pb.Body == nil {
				return py.Float(0), nil
			}
			v := pb.Body.GetLinearVelocity()
			switch axis {
			case "x":
				return py.Float(v.X), nil
			case "y":
				return py.Float(v.Y), nil
			default:
				return py.Float(0), nil
			}
		}, 0, "get_velocity(axis) -> float -- read the physics body linear velocity; axis: 'x' or 'y'")

	m.Globals["apply_linear_impulse_to_center"] = py.MustNewMethod("apply_linear_impulse_to_center",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 2 {
				return nil, py.ExceptionNewf(py.TypeError, "apply_linear_impulse_to_center() requires 2 arguments")
			}
			ix, err := pyToFloat64(args[0])
			if err != nil {
				return nil, err
			}
			iy, err := pyToFloat64(args[1])
			if err != nil {
				return nil, err
			}
			if pb := go_.PhysicsBody(); pb != nil && pb.Body != nil {
				pb.Body.ApplyLinearImpulseToCenter(physics.Vec2{X: ix, Y: iy})
			}
			return py.None, nil
		}, 0, "apply_linear_impulse_to_center(ix, iy) -- apply an instant velocity change (game units/s) at the body center, for dynamic bodies (e.g. a jump)")

	m.Globals["play_animation"] = py.MustNewMethod("play_animation",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "play_animation() requires 1 argument")
			}
			name, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			if anim := go_.Animator(); anim != nil {
				anim.Play(name)
			}
			return py.None, nil
		}, 0, "play_animation(name) -- switch to the named animation")

	m.Globals["reset_animation"] = py.MustNewMethod("reset_animation",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "reset_animation() requires 1 argument")
			}
			name, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			go_.ResetAnimation(name)
			return py.None, nil
		}, 0, "reset_animation(name) -- reset a one-shot animation's frame counter")

	m.Globals["current_animation"] = py.MustNewMethod("current_animation",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if anim := go_.Animator(); anim != nil {
				return py.String(anim.Current), nil
			}
			return py.String(""), nil
		}, 0, "current_animation() -> str -- return the name of the playing animation")

	m.Globals["animation_finished"] = py.MustNewMethod("animation_finished",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			anim := go_.Animator()
			if anim == nil || anim.Current == "" {
				return py.False, nil
			}
			key := "spritesheet:" + anim.Current
			c := go_.GetComponent(key)
			if c == nil {
				return py.False, nil
			}
			if s, ok := c.(*object.Spritesheet); ok {
				return py.NewBool(s.Finished()), nil
			}
			return py.False, nil
		}, 0, "animation_finished() -> bool -- true when a one-shot animation has completed")

	m.Globals["get_intent"] = py.MustNewMethod("get_intent",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "get_intent() requires 1 argument")
			}
			key, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			c := go_.GetComponent("intent_buffer")
			if c == nil {
				return pyIntentZero(key), nil
			}
			ib, ok := c.(*object.IntentBuffer)
			if !ok {
				return pyIntentZero(key), nil
			}
			switch key {
			case "move_x":
				return py.Float(ib.PendingMoveX), nil
			case "move_y":
				return py.Float(ib.PendingMoveY), nil
			default:
				return pyIntentZero(key), nil
			}
		}, 0, "get_intent(key) -> float -- read input intent; keys: 'move_x', 'move_y'")

	m.Globals["get_name"] = py.MustNewMethod("get_name",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			return py.String(go_.Name), nil
		}, 0, "get_name() -> str -- return the entity name")

	m.Globals["get_position"] = py.MustNewMethod("get_position",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "get_position() requires 1 argument")
			}
			axis, err := pyToString(args[0])
			if err != nil {
				return nil, err
			}
			t := go_.Transform()
			if t == nil {
				return py.Float(0), nil
			}
			switch axis {
			case "x":
				return py.Float(t.X), nil
			case "y":
				return py.Float(t.Y), nil
			default:
				return py.Float(0), nil
			}
		}, 0, "get_position(axis) -> float -- read position; axis: 'x' or 'y'")

	m.Globals["set_facing"] = py.MustNewMethod("set_facing",
		func(_ py.Object, args py.Tuple) (py.Object, error) {
			if len(args) < 1 {
				return nil, py.ExceptionNewf(py.TypeError, "set_facing() requires 1 argument")
			}
			dirX, err := pyToFloat64(args[0])
			if err != nil {
				return nil, err
			}
			if t := go_.Transform(); t != nil {
				if dirX < 0 {
					t.ScaleX = -1
				} else {
					t.ScaleX = 1
				}
			}
			return py.None, nil
		}, 0, "set_facing(dir_x) -- flips Transform.ScaleX to -1/1 based on the sign of dir_x; Sprite/Spritesheet/Block all mirror on ScaleX < 0")

	return m
}

// pyIntentZero returns the zero value for intent keys that have numeric semantics.
func pyIntentZero(key string) py.Object {
	switch key {
	case "move_x", "move_y":
		return py.Float(0)
	default:
		return py.False
	}
}
