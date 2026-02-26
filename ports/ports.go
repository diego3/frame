package ports

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"goengine/config"
	"goengine/ui"
	"golang.org/x/image/font"
)

// AssetLoader loads and caches game assets. Implemented by resource.Manager.
type AssetLoader interface {
	LoadImage(path string) (*ebiten.Image, error)
	LoadFont(path string) error
	GetFace(path string, size float64) (font.Face, error)
	LoadAudio(path string) error
	NewAudioPlayer(path string) (*audio.Player, error)
	Clear()
}

// UIRoot is the root UI container. Implemented by ui.Container.
// Update receives layout dimensions and handles cursor conversion and input internally.
// Draw uses the cursor state computed in Update (no cursor args).
type UIRoot interface {
	Update(layoutWidth, layoutHeight int)
	Draw(screen *ebiten.Image, face font.Face)
	AddButton(b *ui.Button)
}

// SceneSwitcher is used by scenes to request a switch to another scene by id.
// Implemented by the game loop; passed to Scene.Setup so scenes can call SwitchTo(id).
type SceneSwitcher interface {
	SwitchTo(sceneID string) error
}

// Scene represents a game screen (e.g. main menu). Setup is called once with dependencies.
// switcher may be nil only when the scene manager is not used.
type Scene interface {
	Setup(cfg *config.Config, loader AssetLoader, ui UIRoot, switcher SceneSwitcher) error
	Update(dt float64)
	Draw(screen *ebiten.Image)
	UIFace() font.Face
}
