//go:build js

package engine

// watchSignals is a no-op on WASM: OS signals are not available in the browser.
// Shutdown is triggered via the QuitRequested event from Lua or UI instead.
func watchSignals(_ chan struct{}) {}
