package object

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

// countingComponent is a minimal Updater/Drawer test double: it records how many times Update and
// Draw were called, so tests can verify Manager skips prototypes without needing real rendering.
type countingComponent struct {
	updates int
	draws   int
}

func (c *countingComponent) Type() string { return "counting" }
func (c *countingComponent) Clone() Component {
	clone := *c
	return &clone
}
func (c *countingComponent) Update(dt float64)                               { c.updates++ }
func (c *countingComponent) Draw(screen *ebiten.Image, transform *Transform) { c.draws++ }

func TestManager_SkipsPrototypesInUpdateAndDraw(t *testing.T) {
	mgr := NewManager()

	real := NewGameObject("real")
	realComp := &countingComponent{}
	real.AddComponent(realComp)
	mgr.Add(real)

	proto := NewGameObject("proto")
	proto.IsPrototype = true
	protoComp := &countingComponent{}
	proto.AddComponent(protoComp)
	mgr.Add(proto)

	mgr.Update(1.0 / 60.0)
	mgr.Draw(nil)

	assert.Equal(t, 1, realComp.updates, "non-prototype should be updated")
	assert.Equal(t, 0, protoComp.updates, "prototype should never be updated")

	t.Run("still reachable via Find/Objects for cloning", func(t *testing.T) {
		assert.Same(t, proto, mgr.Find("proto"))
		assert.Len(t, mgr.Objects(), 2, "Objects() is a raw accessor; prototypes stay reachable for Clone")
	})
}
