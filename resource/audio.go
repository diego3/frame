package resource

import (
	"bytes"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// LoadAudio loads a WAV file from path and caches its bytes. Use NewAudioPlayer to get a playable player.
// Implements ports.AudioLoader.
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
// Implements ports.AudioLoader.
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
