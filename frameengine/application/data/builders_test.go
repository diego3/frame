package data

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goengine/frameengine/object"
	"goengine/frameengine/physics"
	"goengine/frameengine/ports"
)

// fakeImageLoader is a minimal ports.ImageLoader for builder tests that touch images.
type fakeImageLoader struct {
	err error
}

func (f *fakeImageLoader) LoadImage(path string) (*ebiten.Image, error) {
	if f.err != nil {
		return nil, f.err
	}
	return ebiten.NewImage(4, 4), nil
}

var _ ports.ImageLoader = (*fakeImageLoader)(nil)

func TestBuildTransform(t *testing.T) {
	t.Run("defaults scale to 1 when omitted or zero", func(t *testing.T) {
		c, err := buildTransform(map[string]interface{}{"x": 10.0, "y": 20.0}, nil)
		require.NoError(t, err)
		tr := c.(*object.Transform)
		assert.Equal(t, 10.0, tr.X)
		assert.Equal(t, 20.0, tr.Y)
		assert.Equal(t, 1.0, tr.ScaleX)
		assert.Equal(t, 1.0, tr.ScaleY)
	})

	t.Run("keeps explicit scale", func(t *testing.T) {
		c, err := buildTransform(map[string]interface{}{"scale_x": 2.0, "scale_y": 3.0}, nil)
		require.NoError(t, err)
		tr := c.(*object.Transform)
		assert.Equal(t, 2.0, tr.ScaleX)
		assert.Equal(t, 3.0, tr.ScaleY)
	})
}

func TestBuildSprite(t *testing.T) {
	t.Run("missing image path is an error", func(t *testing.T) {
		_, err := buildSprite(map[string]interface{}{}, &fakeImageLoader{})
		assert.Error(t, err)
	})

	t.Run("empty image path is an error", func(t *testing.T) {
		_, err := buildSprite(map[string]interface{}{"image": ""}, &fakeImageLoader{})
		assert.Error(t, err)
	})

	t.Run("loader error propagates", func(t *testing.T) {
		boom := assert.AnError
		_, err := buildSprite(map[string]interface{}{"image": "x.png"}, &fakeImageLoader{err: boom})
		assert.ErrorIs(t, err, boom)
	})

	t.Run("success sets image and layer", func(t *testing.T) {
		c, err := buildSprite(map[string]interface{}{"image": "x.png", "layer": 2.0}, &fakeImageLoader{})
		require.NoError(t, err)
		sprite := c.(*object.Sprite)
		assert.NotNil(t, sprite.Image)
		assert.Equal(t, 2, sprite.Layer)
	})
}

func TestBuildParallaxLayer(t *testing.T) {
	t.Run("missing image path is an error", func(t *testing.T) {
		_, err := buildParallaxLayer(map[string]interface{}{}, &fakeImageLoader{})
		assert.Error(t, err)
	})

	t.Run("loader error propagates", func(t *testing.T) {
		boom := assert.AnError
		_, err := buildParallaxLayer(map[string]interface{}{"image": "sky.png"}, &fakeImageLoader{err: boom})
		assert.ErrorIs(t, err, boom)
	})

	t.Run("scroll_factor defaults to 1 and repeat to false when omitted", func(t *testing.T) {
		c, err := buildParallaxLayer(map[string]interface{}{"image": "sky.png"}, &fakeImageLoader{})
		require.NoError(t, err)
		layer := c.(*object.ParallaxLayer)
		assert.NotNil(t, layer.Image)
		assert.Equal(t, 1.0, layer.ScrollFactor)
		assert.False(t, layer.Repeat)
	})

	t.Run("explicit scroll_factor and repeat are kept, including a zero factor", func(t *testing.T) {
		c, err := buildParallaxLayer(map[string]interface{}{
			"image": "sky.png", "scroll_factor": 0.0, "repeat": true,
		}, &fakeImageLoader{})
		require.NoError(t, err)
		layer := c.(*object.ParallaxLayer)
		assert.Equal(t, 0.0, layer.ScrollFactor)
		assert.True(t, layer.Repeat)
	})
}

func TestBuildSpritesheet(t *testing.T) {
	t.Run("missing frame_width is an error", func(t *testing.T) {
		_, err := buildSpritesheet(map[string]interface{}{
			"image": "x.png", "frame_height": 10.0,
		}, &fakeImageLoader{})
		assert.Error(t, err)
	})

	t.Run("missing frame_height is an error", func(t *testing.T) {
		_, err := buildSpritesheet(map[string]interface{}{
			"image": "x.png", "frame_width": 10.0,
		}, &fakeImageLoader{})
		assert.Error(t, err)
	})

	t.Run("defaults loop to true and frame_index to 0", func(t *testing.T) {
		c, err := buildSpritesheet(map[string]interface{}{
			"image": "x.png", "frame_width": 10.0, "frame_height": 10.0,
		}, &fakeImageLoader{})
		require.NoError(t, err)
		sheet := c.(*object.Spritesheet)
		assert.True(t, sheet.Loop)
		assert.Equal(t, 0, sheet.FrameIndex)
	})

	t.Run("negative frame_index clamps to 0", func(t *testing.T) {
		c, err := buildSpritesheet(map[string]interface{}{
			"image": "x.png", "frame_width": 10.0, "frame_height": 10.0, "frame_index": -5.0,
		}, &fakeImageLoader{})
		require.NoError(t, err)
		sheet := c.(*object.Spritesheet)
		assert.Equal(t, 0, sheet.FrameIndex)
	})

	t.Run("loop: false is respected", func(t *testing.T) {
		c, err := buildSpritesheet(map[string]interface{}{
			"image": "x.png", "frame_width": 10.0, "frame_height": 10.0, "loop": false,
		}, &fakeImageLoader{})
		require.NoError(t, err)
		sheet := c.(*object.Spritesheet)
		assert.False(t, sheet.Loop)
	})
}

func TestBuildAnimator(t *testing.T) {
	c, err := buildAnimator(map[string]interface{}{"current": "idle"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "idle", c.(*object.Animator).Current)
}

func TestBuildBlock(t *testing.T) {
	t.Run("defaults width and height when non-positive", func(t *testing.T) {
		c, err := buildBlock(map[string]interface{}{"width": 0.0, "height": -1.0}, nil)
		require.NoError(t, err)
		blk := c.(*object.Block)
		assert.Equal(t, 64.0, blk.Width)
		assert.Equal(t, 64.0, blk.Height)
	})

	t.Run("no color params leaves Color nil (Draw falls back to DefaultBlockColor)", func(t *testing.T) {
		c, err := buildBlock(map[string]interface{}{}, nil)
		require.NoError(t, err)
		blk := c.(*object.Block)
		assert.Nil(t, blk.Color)
	})

	t.Run("color components clamp to 0-255 and alpha defaults to 255", func(t *testing.T) {
		c, err := buildBlock(map[string]interface{}{
			"color_r": -10.0, "color_g": 300.0, "color_b": 128.0, "color_a": 0.0,
		}, nil)
		require.NoError(t, err)
		blk := c.(*object.Block)
		assert.Equal(t, color.RGBA{R: 0, G: 255, B: 128, A: 255}, blk.Color)
	})

	t.Run("explicit alpha in range is kept", func(t *testing.T) {
		c, err := buildBlock(map[string]interface{}{
			"color_r": 10.0, "color_a": 128.0,
		}, nil)
		require.NoError(t, err)
		blk := c.(*object.Block)
		rgba, ok := blk.Color.(color.RGBA)
		require.True(t, ok)
		assert.Equal(t, uint8(128), rgba.A)
	})
}

func TestBuildBall(t *testing.T) {
	t.Run("defaults radius when non-positive", func(t *testing.T) {
		c, err := buildBall(map[string]interface{}{"radius": 0.0}, nil)
		require.NoError(t, err)
		assert.Equal(t, 24.0, c.(*object.Ball).Radius)
	})

	t.Run("color components clamp to 0-255", func(t *testing.T) {
		c, err := buildBall(map[string]interface{}{
			"color_r": -1.0, "color_g": 999.0, "color_b": 50.0, "color_a": 999.0,
		}, nil)
		require.NoError(t, err)
		ball := c.(*object.Ball)
		assert.Equal(t, color.RGBA{R: 0, G: 255, B: 50, A: 255}, ball.Color)
	})
}

func TestBuildPhysicsBody(t *testing.T) {
	tests := []struct {
		name     string
		bodyType string
		want     physics.BodyType
	}{
		{"kinematic maps to BodyKinematic", "kinematic", physics.BodyKinematic},
		{"dynamic maps to BodyDynamic", "dynamic", physics.BodyDynamic},
		{"static maps to BodyStatic", "static", physics.BodyStatic},
		{"unknown maps to BodyStatic", "bogus", physics.BodyStatic},
		{"missing maps to BodyStatic", "", physics.BodyStatic},
		{"is case-insensitive", "KINEMATIC", physics.BodyKinematic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{}
			if tt.bodyType != "" {
				params["body_type"] = tt.bodyType
			}
			c, err := buildPhysicsBody(params, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, c.(*object.PhysicsBody).BodyType)
		})
	}

	t.Run("defaults width/height when non-positive", func(t *testing.T) {
		c, err := buildPhysicsBody(map[string]interface{}{"width": 0.0, "height": 0.0}, nil)
		require.NoError(t, err)
		pb := c.(*object.PhysicsBody)
		assert.Equal(t, 64.0, pb.Width)
		assert.Equal(t, 64.0, pb.Height)
	})

	t.Run("shape defaults to box, circle is opt-in", func(t *testing.T) {
		c, err := buildPhysicsBody(map[string]interface{}{}, nil)
		require.NoError(t, err)
		assert.Equal(t, physics.ShapeBox, c.(*object.PhysicsBody).Shape)

		c, err = buildPhysicsBody(map[string]interface{}{"shape": "circle"}, nil)
		require.NoError(t, err)
		assert.Equal(t, physics.ShapeCircle, c.(*object.PhysicsBody).Shape)
	})
}

func TestBuildScript(t *testing.T) {
	t.Run("missing path is an error", func(t *testing.T) {
		_, err := buildScript(map[string]interface{}{}, nil)
		assert.Error(t, err)
	})

	t.Run("empty path is an error", func(t *testing.T) {
		_, err := buildScript(map[string]interface{}{"path": ""}, nil)
		assert.Error(t, err)
	})

	t.Run("defaults update_func to update", func(t *testing.T) {
		c, err := buildScript(map[string]interface{}{"path": "scripts/x.lua"}, nil)
		require.NoError(t, err)
		s := c.(*object.Script)
		assert.Equal(t, "scripts/x.lua", s.Path)
		assert.Equal(t, "update", s.UpdateFuncName)
	})

	t.Run("keeps explicit update_func", func(t *testing.T) {
		c, err := buildScript(map[string]interface{}{"path": "scripts/x.lua", "update_func": "tick"}, nil)
		require.NoError(t, err)
		assert.Equal(t, "tick", c.(*object.Script).UpdateFuncName)
	})
}

func TestBuildIntentBuffer(t *testing.T) {
	c, err := buildIntentBuffer(nil, nil)
	require.NoError(t, err)
	assert.IsType(t, &object.IntentBuffer{}, c)
}
