package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	assert.Equal(t, "games/demo1", cfg.GameRoot)
	assert.Equal(t, 640, cfg.Window.Width)
	assert.Equal(t, 480, cfg.Window.Height)
	assert.Equal(t, 320, cfg.Layout.Width)
	assert.Equal(t, 240, cfg.Layout.Height)
	assert.Equal(t, "assets/ShadeBlue-2OozX.ttf", cfg.Assets.FontPath)
	assert.Equal(t, float64(24), cfg.Assets.FontSize)
	assert.Equal(t, "assets/click.wav", cfg.Assets.ClickSound)
	assert.Equal(t, "scenes/main_menu.yaml", cfg.Assets.ScenePath)
	assert.Equal(t, float64(400), cfg.Physics.GravityY)
	assert.Equal(t, float64(64), cfg.Physics.PixelScale)
	assert.Equal(t, map[string]string{"main_menu": "main_menu"}, cfg.Scenes)
	assert.Equal(t, "main_menu", cfg.InitialScene)
}

func TestLoad_missingFile_returnsDefault(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does_not_exist.yaml"))

	require.NoError(t, err)
	assert.Equal(t, Default(), cfg)
}

func TestLoad_invalidYAML_returnsError(t *testing.T) {
	path := writeTempConfig(t, "window: [this is not valid: yaml")

	_, err := Load(path)

	assert.Error(t, err)
}

func TestLoad_setsGameRootFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`window: {width: 800, height: 600}`), 0o644))

	cfg, err := Load(path)

	require.NoError(t, err)
	assert.Equal(t, dir, cfg.GameRoot)
}

func TestLoad_overridesDefaultsFromYAML(t *testing.T) {
	path := writeTempConfig(t, `
window:
  width: 1024
  height: 768
  title: "Custom"
assets:
  font_path: "assets/custom.ttf"
  scene_path: "scenes/level1.yaml"
scenes:
  main_menu: main_menu
  level1: main_menu
initial_scene: level1
`)

	cfg, err := Load(path)

	require.NoError(t, err)
	assert.Equal(t, 1024, cfg.Window.Width)
	assert.Equal(t, 768, cfg.Window.Height)
	assert.Equal(t, "Custom", cfg.Window.Title)
	assert.Equal(t, "assets/custom.ttf", cfg.Assets.FontPath)
	assert.Equal(t, "scenes/level1.yaml", cfg.Assets.ScenePath)
	assert.Equal(t, map[string]string{"main_menu": "main_menu", "level1": "main_menu"}, cfg.Scenes)
	assert.Equal(t, "level1", cfg.InitialScene)
}

// TestLoad_defaultsZeroOrMissingFields verifies every "ensure minimal valid values" branch in Load:
// fields left zero (or empty/missing) in the YAML fall back to the same values Default() would use.
func TestLoad_defaultsZeroOrMissingFields(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		check func(t *testing.T, cfg *Config)
	}{
		{
			name: "negative window size falls back to default",
			yaml: "window: {width: -1, height: -1}",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 640, cfg.Window.Width)
				assert.Equal(t, 480, cfg.Window.Height)
			},
		},
		{
			name: "zero layout size falls back to default",
			yaml: "layout: {width: 0, height: 0}",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 320, cfg.Layout.Width)
				assert.Equal(t, 240, cfg.Layout.Height)
			},
		},
		{
			name: "zero font size falls back to default",
			yaml: "assets: {font_size: 0}",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, float64(24), cfg.Assets.FontSize)
			},
		},
		{
			name: "empty font path and click sound fall back to default",
			yaml: "assets: {font_path: \"\", click_sound: \"\"}",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "assets/ShadeBlue-2OozX.ttf", cfg.Assets.FontPath)
				assert.Equal(t, "assets/click.wav", cfg.Assets.ClickSound)
			},
		},
		{
			name: "non-positive pixel scale falls back to default",
			yaml: "physics: {pixel_scale: -5}",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, float64(64), cfg.Physics.PixelScale)
			},
		},
		{
			name: "missing scenes falls back to default main_menu entry",
			yaml: "window: {width: 800}",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, map[string]string{"main_menu": "main_menu"}, cfg.Scenes)
			},
		},
		{
			name: "missing initial_scene falls back to main_menu",
			yaml: "window: {width: 800}",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "main_menu", cfg.InitialScene)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.yaml)

			cfg, err := Load(path)

			require.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}
