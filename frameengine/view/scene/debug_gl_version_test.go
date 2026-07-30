//go:build !js

package scene

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenGLVersionString_neverEmpty(t *testing.T) {
	// glxinfo may or may not be present/working in the test environment -- either way this must
	// resolve to something displayable ("n/a" as the documented fallback) and never panic or hang.
	got := openGLVersionString()
	assert.NotEmpty(t, got)

	// sync.Once-cached: a second call must return the identical value without re-invoking glxinfo.
	assert.Equal(t, got, openGLVersionString())
}
