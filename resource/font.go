package resource

import (
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// LoadFont loads a TTF or OTF file from path and caches it. Use GetFace to get a font.Face at a given size.
// Implements ports.FontLoader.
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
// Implements ports.FontLoader.
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
