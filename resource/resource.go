package resource

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const defaultSampleRate = 44100

// Manager loads and caches game assets. Safe for concurrent use.
// Root is prepended to relative paths when set (e.g. "games/demo1").
type Manager struct {
	mu         sync.RWMutex
	root       string
	images     map[string]*ebiten.Image
	audioBytes map[string][]byte
	audioCtx   *audio.Context
	fonts      map[string]*opentype.Font
	faces      map[string]font.Face
}

// SetRoot sets the base path for relative asset paths. Paths passed to Load* are joined with root.
func (m *Manager) SetRoot(root string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.root = root
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

// LoadImage loads an image from path and caches it. Subsequent calls with the same path return the cached image.
func (m *Manager) LoadImage(path string) (*ebiten.Image, error) {
	resolved := m.resolve(path)
	m.mu.RLock()
	img, ok := m.images[resolved]
	m.mu.RUnlock()
	if ok {
		return img, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if img, ok = m.images[resolved]; ok {
		return img, nil
	}

	img, _, err := ebitenutil.NewImageFromFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("load image %q: %w", resolved, err)
	}
	m.images[resolved] = img
	return img, nil
}

// LoadAudio loads a WAV file from path and caches its bytes. Use NewAudioPlayer to get a playable player.
func (m *Manager) LoadAudio(path string) error {
	resolved := m.resolve(path)
	m.mu.RLock()
	_, ok := m.audioBytes[resolved]
	m.mu.RUnlock()
	if ok {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok = m.audioBytes[resolved]; ok {
		return nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("load audio %q: %w", resolved, err)
	}
	m.audioBytes[resolved] = data
	if m.audioCtx == nil {
		m.audioCtx = audio.NewContext(defaultSampleRate)
	}
	return nil
}

// NewAudioPlayer returns a new audio.Player for the given path. The path must have been loaded with LoadAudio first.
// Caller may call player.Play() and should not close the player; it is consumed when played.
func (m *Manager) NewAudioPlayer(path string) (*audio.Player, error) {
	resolved := m.resolve(path)
	m.mu.RLock()
	data, ok := m.audioBytes[resolved]
	ctx := m.audioCtx
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("audio not loaded: %q (call LoadAudio first)", resolved)
	}
	if ctx == nil {
		return nil, fmt.Errorf("audio context not initialized")
	}

	stream, err := wav.DecodeWithSampleRate(ctx.SampleRate(), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode audio %q: %w", resolved, err)
	}
	player, err := ctx.NewPlayer(stream)
	if err != nil {
		return nil, fmt.Errorf("new player for %q: %w", resolved, err)
	}
	return player, nil
}

// LoadFont loads a TTF or OTF file from path and caches it. Use GetFace to get a font.Face at a given size.
func (m *Manager) LoadFont(path string) error {
	resolved := m.resolve(path)
	m.mu.RLock()
	_, ok := m.fonts[resolved]
	m.mu.RUnlock()
	if ok {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok = m.fonts[resolved]; ok {
		return nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("load font %q: %w", resolved, err)
	}
	fnt, err := opentype.Parse(data)
	if err != nil {
		return fmt.Errorf("parse font %q: %w", resolved, err)
	}
	m.fonts[resolved] = fnt
	return nil
}

// GetFace returns a font.Face for the given path and size in points. The path must have been loaded with LoadFont first.
func (m *Manager) GetFace(path string, size float64) (font.Face, error) {
	resolved := m.resolve(path)
	key := fmt.Sprintf("%s:%g", resolved, size)
	m.mu.RLock()
	f, ok := m.faces[key]
	m.mu.RUnlock()
	if ok {
		return f, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if f, ok = m.faces[key]; ok {
		return f, nil
	}

	fnt, ok := m.fonts[resolved]
	if !ok {
		return nil, fmt.Errorf("font not loaded: %q (call LoadFont first)", resolved)
	}
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{Size: size, DPI: 72})
	if err != nil {
		return nil, fmt.Errorf("new face for %q: %w", resolved, err)
	}
	m.faces[key] = face
	return face, nil
}

// TextToImage renders text with the given face and color to a new *ebiten.Image. Useful for drawing labels or UI.
func TextToImage(face font.Face, text string, clr color.Color) *ebiten.Image {
	dr := &font.Drawer{
		Face: face,
	}
	adv := dr.MeasureString(text)
	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Ceil()
	width := adv.Ceil()
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	dr.Dst = img
	dr.Src = image.NewUniform(clr)
	dr.Dot = fixed.P(0, metrics.Ascent.Ceil())
	dr.DrawString(text)
	return ebiten.NewImageFromImage(img)
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
