package object

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUpdaterDrawer is a minimal Component that is both an Updater and a Drawer, used to observe
// which components Manager.Update/Draw actually touch without depending on real components like
// Spritesheet (which need a real *ebiten.Image to do anything meaningful).
type fakeUpdaterDrawer struct {
	typeName    string
	updateCalls int
	drawCalls   int
}

func (f *fakeUpdaterDrawer) Type() string { return f.typeName }
func (f *fakeUpdaterDrawer) Clone() Component {
	clone := *f
	return &clone
}
func (f *fakeUpdaterDrawer) Update(dt float64)              { f.updateCalls++ }
func (f *fakeUpdaterDrawer) Draw(*ebiten.Image, *Transform) { f.drawCalls++ }

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

func TestManager_AddFindObjects(t *testing.T) {
	m := NewManager()
	g1 := NewGameObject("a")
	g2 := NewGameObject("b")

	m.Add(g1)
	m.Add(g2)
	m.Add(nil) // must be a no-op, not panic or add a nil entry

	assert.Equal(t, []*GameObject{g1, g2}, m.Objects())
	assert.Same(t, g2, m.Find("b"))
	assert.Nil(t, m.Find("missing"))
}

func TestManager_Update_skipsInactiveObjects(t *testing.T) {
	m := NewManager()
	active := NewGameObject("active")
	comp := &fakeUpdaterDrawer{typeName: "fake"}
	active.AddComponent(comp)
	m.Add(active)

	inactive := NewGameObject("inactive")
	inactive.Active = false
	inactiveComp := &fakeUpdaterDrawer{typeName: "fake"}
	inactive.AddComponent(inactiveComp)
	m.Add(inactive)

	m.Update(1.0 / 60.0)

	assert.Equal(t, 1, comp.updateCalls)
	assert.Equal(t, 0, inactiveComp.updateCalls)
}

func TestManager_Update_withoutAnimator_updatesAllComponents(t *testing.T) {
	m := NewManager()
	g := NewGameObject("obj")
	a := &fakeUpdaterDrawer{typeName: "a"}
	b := &fakeUpdaterDrawer{typeName: "b"}
	g.AddComponent(a)
	g.AddComponent(b)
	m.Add(g)

	m.Update(1.0 / 60.0)

	assert.Equal(t, 1, a.updateCalls)
	assert.Equal(t, 1, b.updateCalls)
}

func TestManager_Update_withAnimator_updatesOnlyActiveSpritesheet(t *testing.T) {
	m := NewManager()
	g := NewGameObject("obj")
	g.AddComponent(&Animator{Current: "run"})

	runFrame := &fakeUpdaterDrawer{typeName: "spritesheet:run"}
	idleFrame := &fakeUpdaterDrawer{typeName: "spritesheet:idle"}
	g.components["spritesheet:run"] = runFrame
	g.components["spritesheet:idle"] = idleFrame
	m.Add(g)

	m.Update(1.0 / 60.0)

	assert.Equal(t, 1, runFrame.updateCalls, "the animator's current spritesheet should update")
	assert.Equal(t, 0, idleFrame.updateCalls, "other spritesheets should not update")
}

func TestManager_Draw_skipsInactiveObjects(t *testing.T) {
	m := NewManager()
	active := NewGameObject("active")
	active.AddComponent(NewTransform(0, 0))
	comp := &fakeUpdaterDrawer{typeName: "fake"}
	active.AddComponent(comp)
	m.Add(active)

	inactive := NewGameObject("inactive")
	inactive.Active = false
	inactive.AddComponent(NewTransform(0, 0))
	inactiveComp := &fakeUpdaterDrawer{typeName: "fake"}
	inactive.AddComponent(inactiveComp)
	m.Add(inactive)

	screen := ebiten.NewImage(1, 1)
	m.Draw(screen)

	assert.Equal(t, 1, comp.drawCalls)
	assert.Equal(t, 0, inactiveComp.drawCalls)
}

func TestManager_Draw_withAnimator_drawsOnlyActiveSpritesheet(t *testing.T) {
	m := NewManager()
	g := NewGameObject("obj")
	g.AddComponent(NewTransform(0, 0))
	g.AddComponent(&Animator{Current: "idle"})

	idleFrame := &fakeUpdaterDrawer{typeName: "spritesheet:idle"}
	runFrame := &fakeUpdaterDrawer{typeName: "spritesheet:run"}
	g.components["spritesheet:idle"] = idleFrame
	g.components["spritesheet:run"] = runFrame
	m.Add(g)

	screen := ebiten.NewImage(1, 1)
	m.Draw(screen)

	assert.Equal(t, 1, idleFrame.drawCalls)
	assert.Equal(t, 0, runFrame.drawCalls)
}

func TestManager_Draw_animatorWithNoMatch_drawsNothing(t *testing.T) {
	m := NewManager()
	g := NewGameObject("obj")
	g.AddComponent(NewTransform(0, 0))
	g.AddComponent(&Animator{Current: "missing"})
	other := &fakeUpdaterDrawer{typeName: "other"}
	g.AddComponent(other)
	m.Add(g)

	screen := ebiten.NewImage(1, 1)
	require.NotPanics(t, func() { m.Draw(screen) })

	assert.Equal(t, 0, other.drawCalls, "components must not be drawn as a fallback when the animator's target is missing")
}

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
