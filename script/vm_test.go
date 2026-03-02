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
