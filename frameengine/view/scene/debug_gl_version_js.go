//go:build js

package scene

// openGLVersionString has no implementation under js/wasm: there's no subprocess to shell out to
// (os/exec doesn't exist for GOOS=js) and no equivalent driver-introspection API in the browser
// WebGL context ebiten uses there -- see debug_gl_version.go for the native implementation.
func openGLVersionString() string {
	return "n/a"
}
