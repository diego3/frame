package scene

import (
	"bytes"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"goengine/frameengine/object"
	"goengine/frameengine/script"
)

// fakeScriptEngine is a minimal script.Engine test double: each method's return value (or error)
// is configured directly, and call counts are recorded so tests can assert on retry/log behavior
// without needing a real Lua/Python interpreter.
type fakeScriptEngine struct {
	doFileErr        error
	doFileCalls      int
	callUpdateErr    error
	callUpdateCalls  int
	callOnEventErr   error
	callOnEventCalls int
}

func (f *fakeScriptEngine) DoFile(path string) error {
	f.doFileCalls++
	return f.doFileErr
}

func (f *fakeScriptEngine) DoString(path, src string) error { return nil }

func (f *fakeScriptEngine) RegisterEngineAPI(
	playSound func(string),
	switchScene func(string),
	quit func(),
	emit func(string, map[string]interface{}),
	getEntityPosition func(string, string) (float64, bool),
) {
}

func (f *fakeScriptEngine) CallScriptUpdate(path, funcName string, go_ *object.GameObject, dt float64) error {
	f.callUpdateCalls++
	return f.callUpdateErr
}

func (f *fakeScriptEngine) CallOnEvent(name string, payload map[string]interface{}) error {
	f.callOnEventCalls++
	return f.callOnEventErr
}

func (f *fakeScriptEngine) Close() {}

var _ script.Engine = (*fakeScriptEngine)(nil)

// newTestWorldScene builds a WorldScene with the given engine and a single active GameObject
// carrying a script component at scriptPath, bypassing Setup (which needs a full config/loader/UI
// stack) so updateScripts can be exercised directly, per CLAUDE.md's preference for testing
// against small seams over bootstrapping the full engine.
func newTestWorldScene(engine script.Engine, scriptPath string) *WorldScene {
	go_ := object.NewGameObject("actor")
	go_.AddComponent(&object.Script{Path: scriptPath})
	mgr := object.NewManager()
	mgr.Add(go_)

	return &WorldScene{
		world:               mgr,
		engine:              engine,
		loadedScripts:       make(map[string]bool),
		scriptLoadFailed:    make(map[string]bool),
		scriptUpdateFailing: make(map[string]bool),
	}
}

func TestUpdateScripts_loadFailure_doesNotRetryEveryFrame(t *testing.T) {
	fake := &fakeScriptEngine{doFileErr: errors.New("syntax error")}
	ws := newTestWorldScene(fake, "scripts/broken.py")

	for i := 0; i < 3; i++ {
		ws.updateScripts(1.0 / 60)
	}

	assert.Equal(t, 1, fake.doFileCalls, "a failed load must not be retried every frame")
	assert.Equal(t, 0, fake.callUpdateCalls, "CallScriptUpdate must not run for a script that failed to load")
	assert.True(t, ws.loadedScripts["scripts/broken.py"])
	assert.True(t, ws.scriptLoadFailed["scripts/broken.py"])
}

func TestUpdateScripts_loadFailure_isLogged(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fake := &fakeScriptEngine{doFileErr: errors.New("boom")}
	ws := newTestWorldScene(fake, "scripts/broken.py")
	ws.updateScripts(1.0 / 60)

	assert.Contains(t, buf.String(), "scripts/broken.py")
	assert.Contains(t, buf.String(), "boom")
}

func TestUpdateScripts_updateFailure_logsOnlyOnTransition(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fake := &fakeScriptEngine{callUpdateErr: errors.New("nil pointer")}
	ws := newTestWorldScene(fake, "scripts/flaky.py")

	for i := 0; i < 3; i++ {
		ws.updateScripts(1.0 / 60)
	}

	assert.Equal(t, 3, fake.callUpdateCalls, "CallScriptUpdate keeps being called every frame even while failing")
	assert.True(t, ws.scriptUpdateFailing["scripts/flaky.py"])
	failedLines := bytes.Count(buf.Bytes(), []byte("failed in"))
	assert.Equal(t, 1, failedLines, "a script failing every frame should log once, not once per frame")
}

func TestUpdateScripts_updateFailure_recoveryIsLogged(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fake := &fakeScriptEngine{callUpdateErr: errors.New("nil pointer")}
	ws := newTestWorldScene(fake, "scripts/flaky.py")

	ws.updateScripts(1.0 / 60) // fails
	fake.callUpdateErr = nil
	ws.updateScripts(1.0 / 60) // recovers

	assert.False(t, ws.scriptUpdateFailing["scripts/flaky.py"])
	assert.Contains(t, buf.String(), "recovered")
}
