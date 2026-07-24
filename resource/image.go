package resource

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// LoadImage loads an image from path and caches it. Subsequent calls with the same path return the cached image.
// Implements ports.ImageLoader.
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
