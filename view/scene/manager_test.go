package scene

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font"

	"goengine/application/config"
	"goengine/event"
	"goengine/ports"
)

// fakeScene is a minimal ports.Scene used to observe what Manager.SwitchTo passes to Setup,
// without depending on a real scene implementation (e.g. WorldScene) or its asset loading.
type fakeScene struct {
	setupErr   error
	gotCtx     *ports.SceneContext
	setupCalls int
}

func (f *fakeScene) Setup(ctx *ports.SceneContext) error {
	f.setupCalls++
	f.gotCtx = ctx
	return f.setupErr
}
func (f *fakeScene) Update(dt float64)         {}
func (f *fakeScene) Draw(screen *ebiten.Image) {}
func (f *fakeScene) UIFace() font.Face         { return nil }

func TestManager_SwitchTo_unknownID(t *testing.T) {
	m := NewManager()

	err := m.SwitchTo("missing", &config.Config{}, nil, nil, event.NewBus())

	assert.ErrorContains(t, err, "unknown scene id")
	assert.Nil(t, m.CurrentScene())
}

func TestManager_SwitchTo_factoryError(t *testing.T) {
	m := NewManager()
	boom := errors.New("factory boom")
	m.Register("broken", func() (ports.Scene, error) { return nil, boom })

	err := m.SwitchTo("broken", &config.Config{}, nil, nil, event.NewBus())

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, m.CurrentScene())
}

func TestManager_SwitchTo_setupError_doesNotChangeCurrentScene(t *testing.T) {
	m := NewManager()
	first := &fakeScene{}
	m.Register("first", func() (ports.Scene, error) { return first, nil })
	require.NoError(t, m.SwitchTo("first", &config.Config{}, nil, nil, event.NewBus()))
	require.Same(t, first, m.CurrentScene())

	boom := errors.New("setup boom")
	m.Register("broken", func() (ports.Scene, error) { return &fakeScene{setupErr: boom}, nil })

	err := m.SwitchTo("broken", &config.Config{}, nil, nil, event.NewBus())

	assert.ErrorIs(t, err, boom)
	assert.Same(t, first, m.CurrentScene(), "a failed switch must not replace the current scene")
}

func TestManager_SwitchTo_success_setsCurrentSceneAndBuildsContext(t *testing.T) {
	m := NewManager()
	sc := &fakeScene{}
	m.Register("menu", func() (ports.Scene, error) { return sc, nil })

	cfg := &config.Config{GameRoot: "games/demo1"}
	bus := event.NewBus()

	err := m.SwitchTo("menu", cfg, nil, nil, bus)

	require.NoError(t, err)
	assert.Same(t, sc, m.CurrentScene())
	require.Equal(t, 1, sc.setupCalls)
	require.NotNil(t, sc.gotCtx)
	assert.Same(t, cfg, sc.gotCtx.Config)
	assert.Same(t, bus, sc.gotCtx.Bus)
}

func TestManager_Register_overwritesExistingID(t *testing.T) {
	m := NewManager()
	first := &fakeScene{}
	second := &fakeScene{}
	m.Register("menu", func() (ports.Scene, error) { return first, nil })
	m.Register("menu", func() (ports.Scene, error) { return second, nil })

	require.NoError(t, m.SwitchTo("menu", &config.Config{}, nil, nil, event.NewBus()))

	assert.Same(t, second, m.CurrentScene())
}

func TestManager_CurrentScene_nilBeforeAnySwitch(t *testing.T) {
	m := NewManager()
	assert.Nil(t, m.CurrentScene())
}
