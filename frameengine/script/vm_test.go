package script

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestVM_DoString(t *testing.T) {
	vm := NewVM()
	defer vm.Close()

	// Register dummy engine so script can call engine.quit() without panicking
	called := false
	vm.RegisterEngine("engine", EngineFuncs(
		func(path string) {},
		func(sceneID string) {},
		func() { called = true },
		func(name string, payload map[string]interface{}) {},
		func(name, axis string) (float64, bool) { return 0, false },
	))

	if err := vm.DoString(`engine.quit()`); err != nil {
		t.Fatalf("DoString: %v", err)
	}
	if !called {
		t.Error("engine.quit() did not call Go callback")
	}
}

func TestVM_DoString_script_error(t *testing.T) {
	vm := NewVM()
	defer vm.Close()

	vm.RegisterEngine("engine", EngineFuncs(
		func(string) {},
		func(string) {},
		func() {},
		func(string, map[string]interface{}) {},
		func(string, string) (float64, bool) { return 0, false },
	))

	err := vm.DoString(`syntax error here`)
	if err == nil {
		t.Fatal("expected error for invalid Lua")
	}
}

func TestVM_CallFunc(t *testing.T) {
	vm := NewVM()
	defer vm.Close()

	if err := vm.DoString(`function add(a, b) return a + b end`); err != nil {
		t.Fatalf("DoString: %v", err)
	}

	ret, err := vm.CallFunc("add", lua.LNumber(2), lua.LNumber(3))
	if err != nil {
		t.Fatalf("CallFunc: %v", err)
	}
	if len(ret) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(ret))
	}
	if n, ok := ret[0].(lua.LNumber); !ok || float64(n) != 5 {
		t.Errorf("expected 5, got %v", ret[0])
	}
}

func TestVM_engine_emit(t *testing.T) {
	vm := NewVM()
	defer vm.Close()

	var gotName string
	var gotPayload map[string]interface{}
	vm.RegisterEngine("engine", EngineFuncs(
		func(string) {},
		func(string) {},
		func() {},
		func(name string, payload map[string]interface{}) {
			gotName = name
			gotPayload = payload
		},
		func(name, axis string) (float64, bool) { return 0, false },
	))

	err := vm.DoString(`engine.emit("ItemCollected", { item_id = "sword", count = 1 })`)
	if err != nil {
		t.Fatalf("DoString: %v", err)
	}
	if gotName != "ItemCollected" {
		t.Errorf("got name %q", gotName)
	}
	if gotPayload["item_id"] != "sword" || gotPayload["count"] != float64(1) {
		t.Errorf("got payload %v", gotPayload)
	}
}

func TestVM_CallOnEvent(t *testing.T) {
	vm := NewVM()
	defer vm.Close()

	var gotName string
	if err := vm.DoString(`
		function on_event(name, payload)
			-- store in globals for test to read via Lua
			_event_name = name
			_event_payload = payload
		end
	`); err != nil {
		t.Fatalf("DoString: %v", err)
	}

	err := CallOnEvent(vm.L, "ScoreChanged", map[string]interface{}{"score": float64(100)})
	if err != nil {
		t.Fatalf("CallOnEvent: %v", err)
	}
	gotName = vm.L.GetGlobal("_event_name").String()
	if gotName != "ScoreChanged" {
		t.Errorf("got name %q", gotName)
	}
	tbl, ok := vm.L.GetGlobal("_event_payload").(*lua.LTable)
	if !ok {
		t.Fatal("_event_payload is not a table")
	}
	scoreVal := vm.L.GetField(tbl, "score")
	if num, ok := scoreVal.(lua.LNumber); !ok || float64(num) != 100 {
		t.Errorf("payload.score = %v", scoreVal)
	}
}
