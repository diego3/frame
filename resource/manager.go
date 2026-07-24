// Package resource implements game asset loading (images, audio, fonts) and caching.
// Manager's methods are split by concern across image.go, audio.go, and font.go; Manager
// itself implements ports.AssetLoader by composing the smaller ports.ImageLoader,
// ports.FontLoader, and ports.AudioLoader interfaces.
package resource

import (
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const defaultSampleRate = 44100

// Manager loads and caches game assets. Safe for concurrent use.
// Root is prepended to relative paths when set (e.g. "games/demo1").
// When an fs.FS is set via SetFS, all reads use the embedded FS instead of the OS filesystem
// and the root prefix is not applied (the FS root is already the game root).
type Manager struct {
	mu         sync.RWMutex
	root       string
	fsys       fs.FS
	images     map[string]*ebiten.Image
	audioBytes map[string][]byte
	audioCtx   *audio.Context
	fonts      map[string]*opentype.Font
	faces      map[string]font.Face
}

// SetRoot sets the base path for relative asset paths. Paths passed to Load* are joined with root.
// Has no effect when an fs.FS is set via SetFS (the FS root is already the game root).
func (m *Manager) SetRoot(root string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.root = root
}

// SetFS configures the manager to read all assets from fsys instead of the OS filesystem.
// When set, the root prefix is not applied; paths are used as-is relative to the FS root.
// Use with an embed.FS for WASM builds where the OS filesystem is unavailable.
func (m *Manager) SetFS(fsys fs.FS) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fsys = fsys
}

// NewManager returns a new resource manager.
func NewManager() *Manager {
	return &Manager{
		images:     make(map[string]*ebiten.Image),
		audioBytes: make(map[string][]byte),
		fonts:      make(map[string]*opentype.Font),
		faces:      make(map[string]font.Face),
	}
}

func (m *Manager) resolve(path string) string {
	m.mu.RLock()
	root := m.root
	m.mu.RUnlock()
	if root == "" {
		return path
	}
	return filepath.Join(root, path)
}

// Clear releases all cached resources. Call from game shutdown.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.images = make(map[string]*ebiten.Image)
	m.audioBytes = make(map[string][]byte)
	m.fonts = make(map[string]*opentype.Font)
	m.faces = make(map[string]font.Face)
	// audioCtx left as-is; no Close in standard API
}
