package input

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// keyByName maps normalized key name (lowercase, no spaces) to ebiten.Key.
// Used to parse config input bindings (e.g. "shiftleft", "a", "f3").
var keyByName map[string]ebiten.Key

func init() {
	keyByName = map[string]ebiten.Key{
		"a": ebiten.KeyA, "b": ebiten.KeyB, "c": ebiten.KeyC, "d": ebiten.KeyD, "e": ebiten.KeyE,
		"f": ebiten.KeyF, "g": ebiten.KeyG, "h": ebiten.KeyH, "i": ebiten.KeyI, "j": ebiten.KeyJ,
		"k": ebiten.KeyK, "l": ebiten.KeyL, "m": ebiten.KeyM, "n": ebiten.KeyN, "o": ebiten.KeyO,
		"p": ebiten.KeyP, "q": ebiten.KeyQ, "r": ebiten.KeyR, "s": ebiten.KeyS, "t": ebiten.KeyT,
		"u": ebiten.KeyU, "v": ebiten.KeyV, "w": ebiten.KeyW, "x": ebiten.KeyX, "y": ebiten.KeyY, "z": ebiten.KeyZ,
		"0": ebiten.Key0, "1": ebiten.Key1, "2": ebiten.Key2, "3": ebiten.Key3, "4": ebiten.Key4,
		"5": ebiten.Key5, "6": ebiten.Key6, "7": ebiten.Key7, "8": ebiten.Key8, "9": ebiten.Key9,
		"space": ebiten.KeySpace, "enter": ebiten.KeyEnter, "escape": ebiten.KeyEscape, "tab": ebiten.KeyTab,
		"shiftleft": ebiten.KeyShiftLeft, "shiftright": ebiten.KeyShiftRight,
		"controlleft": ebiten.KeyControlLeft, "controlright": ebiten.KeyControlRight,
		"altleft": ebiten.KeyAltLeft, "altright": ebiten.KeyAltRight,
		"left": ebiten.KeyLeft, "right": ebiten.KeyRight, "up": ebiten.KeyUp, "down": ebiten.KeyDown,
		"f1": ebiten.KeyF1, "f2": ebiten.KeyF2, "f3": ebiten.KeyF3, "f4": ebiten.KeyF4,
		"f5": ebiten.KeyF5, "f6": ebiten.KeyF6, "f7": ebiten.KeyF7, "f8": ebiten.KeyF8,
		"f9": ebiten.KeyF9, "f10": ebiten.KeyF10, "f11": ebiten.KeyF11, "f12": ebiten.KeyF12,
	}
}

// normalizeKeyName returns a lowercase key name with no spaces or hyphens (e.g. "Shift-Left" -> "shiftleft").
func normalizeKeyName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		if r != ' ' && r != '-' && r != '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ParseKey returns the ebiten.Key for the given key name (e.g. "A", "ShiftLeft", "F3").
// Returns false if the name is unknown.
func ParseKey(name string) (ebiten.Key, bool) {
	k, ok := keyByName[normalizeKeyName(name)]
	return k, ok
}

// ParseBindings converts config input section (action -> key name or list of key names) into
// action -> []ebiten.Key. Values can be a single string or YAML list of strings.
// Unknown key names are skipped; at least one valid key per action is required.
func ParseBindings(input map[string]any) (map[string][]ebiten.Key, error) {
	out := make(map[string][]ebiten.Key)
	for action, val := range input {
		var names []string
		switch v := val.(type) {
		case string:
			names = []string{v}
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					names = append(names, s)
				}
			}
		default:
			continue
		}
		var keys []ebiten.Key
		for _, n := range names {
			if k, ok := ParseKey(n); ok {
				keys = append(keys, k)
			}
		}
		if len(keys) > 0 {
			out[action] = keys
		}
	}
	return out, nil
}
