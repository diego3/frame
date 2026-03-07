package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds engine and window settings loaded from YAML.
// GameRoot is the directory of the config file; asset/scene paths are relative to it.
type Config struct {
	GameRoot string  `yaml:"-"` // set from config file path (e.g. "games/demo1")
	Window   Window  `yaml:"window"`
	Layout   Layout  `yaml:"layout"`
	Assets   Assets  `yaml:"assets"`
	Physics  Physics `yaml:"physics"`
}

// Physics holds parameters for the physics simulation (e.g. Box2D).
type Physics struct {
	GravityX   float64 `yaml:"gravity_x"`   // gravity X in game units/s² (usually 0)
	GravityY   float64 `yaml:"gravity_y"`   // gravity Y in game units/s² (positive = down)
	PixelScale float64 `yaml:"pixel_scale"`  // game units per meter (e.g. 64)
}

// Assets holds paths and options for loaded assets.
type Assets struct {
	FontPath   string  `yaml:"font_path"`
	FontSize   float64 `yaml:"font_size"`
	ClickSound string  `yaml:"click_sound"`
	ScenePath  string  `yaml:"scene_path"` // optional: path to scene YAML (objects + components)
}

// Window defines the application window.
type Window struct {
	Width      int    `yaml:"width"`
	Height     int    `yaml:"height"`
	Title      string `yaml:"title"`
	Resizable  bool   `yaml:"resizable"`
	Fullscreen bool   `yaml:"fullscreen"`
}

// Layout defines the logical screen size used for drawing (scaled to window).
type Layout struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

// Default returns a config with sensible defaults. GameRoot is set to "games/demo1".
func Default() *Config {
	return &Config{
		GameRoot: "games/demo1",
		Window: Window{
			Width:  640,
			Height: 480,
			Title:  "GoEngine",
		},
		Layout: Layout{
			Width:  320,
			Height: 240,
		},
		Assets: Assets{
			FontPath:   "assets/ShadeBlue-2OozX.ttf",
			FontSize:   24,
			ClickSound: "assets/click.wav",
			ScenePath:  "scenes/main_menu.yaml",
		},
		Physics: Physics{
			GravityY:   400,
			PixelScale: 64,
		},
	}
}

// Load reads config from path. If the file is missing, returns Default().
// If the file exists but is invalid, returns an error.
// GameRoot is set to the directory of path (e.g. "games/demo1" for "games/demo1/config.yaml").
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}
	cfg := Default()
	cfg.GameRoot = filepath.Dir(path)
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// Ensure minimal valid values
	if cfg.Window.Width <= 0 {
		cfg.Window.Width = 640
	}
	if cfg.Window.Height <= 0 {
		cfg.Window.Height = 480
	}
	if cfg.Layout.Width <= 0 {
		cfg.Layout.Width = 320
	}
	if cfg.Layout.Height <= 0 {
		cfg.Layout.Height = 240
	}
	if cfg.Assets.FontSize <= 0 {
		cfg.Assets.FontSize = 24
	}
	if cfg.Assets.FontPath == "" {
		cfg.Assets.FontPath = "assets/ShadeBlue-2OozX.ttf"
	}
	if cfg.Assets.ClickSound == "" {
		cfg.Assets.ClickSound = "assets/click.wav"
	}
	if cfg.Physics.PixelScale <= 0 {
		cfg.Physics.PixelScale = 64
	}
	return cfg, nil
}
