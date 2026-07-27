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
	RegisterComponentBuilder("projectile", buildProjectile)
	RegisterComponentBuilder("enemy", buildEnemy)
	RegisterComponentBuilder("timer", buildTimer)
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
	sheet := &object.Spritesheet{
		Name:        name,
		Image:       img,
		FrameWidth:  fw,
		FrameHeight: fh,
		Cols:        cols,
		Rows:        rows,
		FrameIndex:  frameIndex,
		FPS:         fps,
		Loop:        loop,
	}
	if _, hasTint := params["tint_r"]; hasTint {
		r, _ := intParam(params, "tint_r")
		g, _ := intParam(params, "tint_g")
		b, _ := intParam(params, "tint_b")
		clamp := func(v int) uint8 {
			if v < 0 {
				return 0
			}
			if v > 255 {
				return 255
			}
			return uint8(v)
		}
		sheet.HasTint = true
		sheet.Tint = color.RGBA{clamp(r), clamp(g), clamp(b), 255}
	}
	return sheet, nil
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
	facingMarker, _ := boolParam(params, "facing_marker")
	blk := &object.Block{Width: width, Height: height, FacingMarker: facingMarker}
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
	fixedRotation, _ := boolParam(params, "fixed_rotation")
	return &object.PhysicsBody{
		Body:          nil,
		BodyType:      bodyType,
		Shape:         shape,
		Width:         width,
		Height:        height,
		Radius:        radius,
		OffsetX:       offsetX,
		OffsetY:       offsetY,
		Density:       density,
		Mass:          mass,
		Restitution:   restitution,
		Friction:      friction,
		FixedRotation: fixedRotation,
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

// buildProjectile builds a Projectile component. vel_x/vel_y are typically left at 0 for a
// "projectile_prototype" template (spawnProjectile recomputes them on every clone from speed and
// the requested direction), but are parsed for consistency and in case a scene ever places a live
// projectile directly in its YAML. speed/spawn_clearance/despawn_margin default to this demo's
// original hardcoded values (360/5/32) when unset or <= 0, matching the zero-means-default
// convention the other builders in this file use (buildBlock, buildBall, buildEnemy, ...).
func buildProjectile(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	velX, _ := floatParam(params, "vel_x")
	velY, _ := floatParam(params, "vel_y")
	damage, _ := floatParam(params, "damage")
	speed, _ := floatParam(params, "speed")
	if speed <= 0 {
		speed = 360
	}
	clearance, _ := floatParam(params, "spawn_clearance")
	if clearance <= 0 {
		clearance = 5
	}
	margin, _ := floatParam(params, "despawn_margin")
	if margin <= 0 {
		margin = 32
	}
	return &object.Projectile{
		VelX: velX, VelY: velY, Damage: damage,
		Speed: speed, SpawnClearance: clearance, DespawnMargin: margin,
	}, nil
}

// buildEnemy builds an Enemy component. hp defaults to 1 (dies on first hit) if unset or <= 0.
func buildEnemy(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	hp, _ := floatParam(params, "hp")
	if hp <= 0 {
		hp = 1
	}
	return &object.Enemy{HP: hp}, nil
}

// buildTimer builds a Timer component. "seconds" is typically left at a placeholder on a
// prototype (e.g. "sphere_prototype") and overwritten by whatever spawns the clone (see
// scene.WorldScene's spawnEntity handling of the "timer_seconds" payload key), same convention as
// buildProjectile.
func buildTimer(params map[string]interface{}, _ ports.ImageLoader) (object.Component, error) {
	seconds, _ := floatParam(params, "seconds")
	return &object.Timer{Remaining: seconds}, nil
}
