package data

import (
	"fmt"
	"image/color"
	"strings"

	"goengine/object"
	"goengine/physics"
	"goengine/ports"
)

func init() {
	RegisterComponentBuilder("transform", buildTransform)
	RegisterComponentBuilder("sprite", buildSprite)
	RegisterComponentBuilder("spritesheet", buildSpritesheet)
	RegisterComponentBuilder("animator", buildAnimator)
	RegisterComponentBuilder("block", buildBlock)
	RegisterComponentBuilder("ball", buildBall)
	RegisterComponentBuilder("physics_body", buildPhysicsBody)
	RegisterComponentBuilder("script", buildScript)
	RegisterComponentBuilder("intent_buffer", buildIntentBuffer)
}

func buildTransform(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	x, _ := floatParam(params, "x")
	y, _ := floatParam(params, "y")
	angle, _ := floatParam(params, "angle")
	scaleX, _ := floatParam(params, "scale_x")
	if scaleX == 0 {
		scaleX = 1
	}
	scaleY, _ := floatParam(params, "scale_y")
	if scaleY == 0 {
		scaleY = 1
	}
	return &object.Transform{
		X: x, Y: y, Angle: angle,
		ScaleX: scaleX, ScaleY: scaleY,
	}, nil
}

func buildSprite(params map[string]interface{}, loader ports.ImageLoader) (object.Component, error) {
	path, err := stringParam(params, "image")
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("sprite: image path required")
	}
	img, err := loader.LoadImage(path)
	if err != nil {
		return nil, err
	}
	layer, _ := intParam(params, "layer")
	return &object.Sprite{Image: img, Layer: layer}, nil
}

func buildSpritesheet(params map[string]interface{}, loader ports.ImageLoader) (object.Component, error) {
	name, _ := stringParam(params, "name")
	path, err := stringParam(params, "image")
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("spritesheet: image path required")
	}
	img, err := loader.LoadImage(path)
	if err != nil {
		return nil, err
	}
	fw, err := intParam(params, "frame_width")
	if err != nil || fw <= 0 {
		return nil, fmt.Errorf("spritesheet: frame_width required and must be > 0")
	}
	fh, err := intParam(params, "frame_height")
	if err != nil || fh <= 0 {
		return nil, fmt.Errorf("spritesheet: frame_height required and must be > 0")
	}
	cols, _ := intParam(params, "cols")
	rows, _ := intParam(params, "rows")
	fps, _ := floatParam(params, "fps")
	frameIndex, _ := intParam(params, "frame_index")
	if frameIndex < 0 {
		frameIndex = 0
	}
	loop := true
	if v, ok := params["loop"].(bool); ok {
		loop = v
	}
	return &object.Spritesheet{
		Name:        name,
		Image:       img,
		FrameWidth:  fw,
		FrameHeight: fh,
		Cols:        cols,
		Rows:        rows,
		FrameIndex:  frameIndex,
		FPS:         fps,
		Loop:        loop,
	}, nil
}

func buildAnimator(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	current, _ := stringParam(params, "current")
	return &object.Animator{Current: current}, nil
}

func buildBlock(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	width, _ := floatParam(params, "width")
	height, _ := floatParam(params, "height")
	if width <= 0 {
		width = 64
	}
	if height <= 0 {
		height = 64
	}
	blk := &object.Block{Width: width, Height: height}
	if _, hasColor := params["color_r"]; hasColor {
		r, _ := intParam(params, "color_r")
		g, _ := intParam(params, "color_g")
		b, _ := intParam(params, "color_b")
		a, _ := intParam(params, "color_a")
		clamp := func(v int) uint8 {
			if v < 0 {
				return 0
			}
			if v > 255 {
				return 255
			}
			return uint8(v)
		}
		if a <= 0 || a > 255 {
			a = 255
		}
		blk.Color = color.RGBA{clamp(r), clamp(g), clamp(b), uint8(a)}
	}
	return blk, nil
}

func buildBall(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	radius, _ := floatParam(params, "radius")
	if radius <= 0 {
		radius = 24
	}
	ball := &object.Ball{Radius: radius}
	if _, hasColor := params["color_r"]; hasColor {
		r, _ := intParam(params, "color_r")
		g, _ := intParam(params, "color_g")
		b, _ := intParam(params, "color_b")
		a, _ := intParam(params, "color_a")
		clamp := func(v int) uint8 {
			if v < 0 {
				return 0
			}
			if v > 255 {
				return 255
			}
			return uint8(v)
		}
		if a <= 0 || a > 255 {
			a = 255
		}
		ball.Color = color.RGBA{clamp(r), clamp(g), clamp(b), uint8(a)}
	}
	return ball, nil
}

func buildPhysicsBody(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	bodyType := physics.BodyStatic
	if s, _ := stringParam(params, "body_type"); s != "" {
		switch strings.ToLower(s) {
		case "kinematic":
			bodyType = physics.BodyKinematic
		case "dynamic":
			bodyType = physics.BodyDynamic
		default:
			bodyType = physics.BodyStatic
		}
	}
	width, _ := floatParam(params, "width")
	height, _ := floatParam(params, "height")
	if width <= 0 {
		width = 64
	}
	if height <= 0 {
		height = 64
	}
	density, _ := floatParam(params, "density")
	mass, _ := floatParam(params, "mass")
	offsetX, _ := floatParam(params, "offset_x")
	offsetY, _ := floatParam(params, "offset_y")
	restitution, _ := floatParam(params, "restitution")
	friction, _ := floatParam(params, "friction")
	radius, _ := floatParam(params, "radius")
	shape := physics.ShapeBox
	if s, _ := stringParam(params, "shape"); strings.ToLower(s) == "circle" {
		shape = physics.ShapeCircle
	}
	return &object.PhysicsBody{
		Body:        nil,
		BodyType:    bodyType,
		Shape:       shape,
		Width:       width,
		Height:      height,
		Radius:      radius,
		OffsetX:     offsetX,
		OffsetY:     offsetY,
		Density:     density,
		Mass:        mass,
		Restitution: restitution,
		Friction:    friction,
	}, nil
}

func buildScript(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	path, err := stringParam(params, "path")
	if err != nil || path == "" {
		return nil, fmt.Errorf("script: path required")
	}
	updateFunc, _ := stringParam(params, "update_func")
	if updateFunc == "" {
		updateFunc = "update"
	}
	return &object.Script{
		Path:           path,
		UpdateFuncName: updateFunc,
	}, nil
}

func buildIntentBuffer(_ map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	return &object.IntentBuffer{}, nil
}
