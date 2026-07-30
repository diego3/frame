package ports

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"golang.org/x/image/font"

	"goengine/frameengine/application/config"
	"goengine/frameengine/event"
	"goengine/frameengine/view/ui"
)

// ImageLoader loads and caches images. A subset of AssetLoader for consumers (e.g. component
// builders) that only need images.
type ImageLoader interface {
	LoadImage(path string) (*ebiten.Image, error)
}

// FontLoader loads fonts and font faces. A subset of AssetLoader for consumers that only need text rendering.
type FontLoader interface {
	LoadFont(path string) error
	GetFace(path string, size float64) (font.Face, error)
}

// AudioLoader loads audio and creates players. A subset of AssetLoader for consumers that only need sound.
type AudioLoader interface {
	LoadAudio(path string) error
	NewAudioPlayer(path string) (*audio.Player, error)
}

// AssetLoader loads and caches game assets. Implemented by resource.Manager.
// It composes the smaller ImageLoader/FontLoader/AudioLoader ports; depend on those instead when
// a consumer only needs one kind of asset (e.g. component builders only need ImageLoader).
// SetRoot sets the base path for relative asset paths (e.g. game folder "games/demo1").
type AssetLoader interface {
	SetRoot(root string)
	ImageLoader
	FontLoader
	AudioLoader
	Clear()
}

// UIRoot is the root UI container. Implemented by ui.Container.
// Update receives layout dimensions and handles cursor conversion and input internally.
// Draw uses the cursor state computed in Update (no cursor args).
type UIRoot interface {
	Update(layoutWidth, layoutHeight int)
	Draw(screen *ebiten.Image, face font.Face)
	AddElement(e ui.Element)
}

// SceneContext bundles the dependencies a Scene needs in Setup. Adding a new dependency only
// means adding a field here, not changing every Scene implementation's Setup signature.
type SceneContext struct {
	Config *config.Config
	Loader AssetLoader
	UI     UIRoot
	// Bus lets scenes emit intents (e.g. SceneChangeRequested, QuitRequested) and subscribe to
	// intent/state events. For emission-only dependencies, prefer event.Emitter or
	// event.IntentEmitter (see event package).
	Bus *event.Bus
}

// Scene represents a game screen (e.g. main menu). Setup is called once with dependencies.
type Scene interface {
	Setup(ctx *SceneContext) error
	Update(dt float64)
	Draw(screen *ebiten.Image)
	UIFace() font.Face
}
