package resource

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
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
type Manager struct {
	mu         sync.RWMutex
	images     map[string]*ebiten.Image
	audioBytes map[string][]byte
	audioCtx   *audio.Context
	fonts      map[string]*opentype.Font
	faces      map[string]font.Face
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

// LoadImage loads an image from path and caches it. Subsequent calls with the same path return the cached image.
func (m *Manager) LoadImage(path string) (*ebiten.Image, error) {
	m.mu.RLock()
	img, ok := m.images[path]
	m.mu.RUnlock()
	if ok {
		return img, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock
	if img, ok = m.images[path]; ok {
		return img, nil
	}

	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load image %q: %w", path, err)
	}
	m.images[path] = img
	return img, nil
}

// LoadAudio loads a WAV file from path and caches its bytes. Use NewAudioPlayer to get a playable player.
func (m *Manager) LoadAudio(path string) error {
	m.mu.RLock()
	_, ok := m.audioBytes[path]
	m.mu.RUnlock()
	if ok {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok = m.audioBytes[path]; ok {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load audio %q: %w", path, err)
	}
	m.audioBytes[path] = data
	if m.audioCtx == nil {
		m.audioCtx = audio.NewContext(defaultSampleRate)
	}
	return nil
}

// NewAudioPlayer returns a new audio.Player for the given path. The path must have been loaded with LoadAudio first.
// Caller may call player.Play() and should not close the player; it is consumed when played.
func (m *Manager) NewAudioPlayer(path string) (*audio.Player, error) {
	m.mu.RLock()
	data, ok := m.audioBytes[path]
	ctx := m.audioCtx
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("audio not loaded: %q (call LoadAudio first)", path)
	}
	if ctx == nil {
		return nil, fmt.Errorf("audio context not initialized")
	}

	stream, err := wav.DecodeWithSampleRate(ctx.SampleRate(), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode audio %q: %w", path, err)
	}
	player, err := ctx.NewPlayer(stream)
	if err != nil {
		return nil, fmt.Errorf("new player for %q: %w", path, err)
	}
	return player, nil
}

// LoadFont loads a TTF or OTF file from path and caches it. Use GetFace to get a font.Face at a given size.
func (m *Manager) LoadFont(path string) error {
	m.mu.RLock()
	_, ok := m.fonts[path]
	m.mu.RUnlock()
	if ok {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok = m.fonts[path]; ok {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load font %q: %w", path, err)
	}
	fnt, err := opentype.Parse(data)
	if err != nil {
		return fmt.Errorf("parse font %q: %w", path, err)
	}
	m.fonts[path] = fnt
	return nil
}

// GetFace returns a font.Face for the given path and size in points. The path must have been loaded with LoadFont first.
func (m *Manager) GetFace(path string, size float64) (font.Face, error) {
	key := fmt.Sprintf("%s:%g", path, size)
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

	fnt, ok := m.fonts[path]
	if !ok {
		return nil, fmt.Errorf("font not loaded: %q (call LoadFont first)", path)
	}
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{Size: size, DPI: 72})
	if err != nil {
		return nil, fmt.Errorf("new face for %q: %w", path, err)
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
