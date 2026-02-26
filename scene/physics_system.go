package scene

import (
	"fmt"
	"image/color"
	"log"

	"goengine/object"
	"goengine/physics"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// kinematicEntry stores a body, size, and offset for syncing position back to Transform.
type kinematicEntry struct {
	body        physics.Body
	width       float64
	height      float64
	offsetX     float64
	offsetY     float64
}

// staticDebugInfo stores position/size of a static body for debug draw and logging.
type staticDebugInfo struct {
	name   string
	center physics.Vec2
	width  float64
	height float64
}

// PhysicsSystem runs the physics world and syncs kinematic body positions to object Transforms.
// Only this and the physics/box2d package need to know about the concrete physics backend.
type PhysicsSystem struct {
	world           physics.World
	kinematicBodies map[string]kinematicEntry
	staticDebug     []staticDebugInfo
}

// NewPhysicsSystem creates a system that steps the given physics world and syncs named bodies to the object world.
func NewPhysicsSystem(world physics.World) *PhysicsSystem {
	return &PhysicsSystem{
		world:           world,
		kinematicBodies: make(map[string]kinematicEntry),
	}
}

// CreateKinematicBody creates a kinematic body for the named object and adds a PhysicsBody component.
// Position is taken from the object's Transform (top-left); body uses center. Width/Height in game units.
func (s *PhysicsSystem) CreateKinematicBody(objectWorld *object.World, name string, width, height float64) error {
	obj := objectWorld.Find(name)
	if obj == nil {
		return nil
	}
	t := obj.Transform()
	if t == nil {
		return nil
	}
	center := physics.Vec2{
		X: t.X + width*0.5,
		Y: t.Y + height*0.5,
	}
	body, err := s.world.CreateBody(physics.BodyDef{
		Position: center,
		Width:    width,
		Height:   height,
		Type:     physics.BodyKinematic,
	})
	if err != nil {
		return err
	}
	obj.AddComponent(&object.PhysicsBody{Body: body, Width: width, Height: height})
	s.kinematicBodies[name] = kinematicEntry{body: body, width: width, height: height, offsetX: 0, offsetY: 0}
	return nil
}

// CreateStaticBody adds a static body at the given center position and size. No sync (static bodies don't move).
func (s *PhysicsSystem) CreateStaticBody(center physics.Vec2, width, height float64) error {
	_, err := s.world.CreateBody(physics.BodyDef{
		Position: center,
		Width:    width,
		Height:   height,
		Type:     physics.BodyStatic,
	})
	return err
}

// InitFromWorld creates physics bodies for all objects that have a PhysicsBody component with Body==nil
// (e.g. loaded from YAML). Uses Transform for position (top-left -> center). For static bodies with
// Width/Height zero, uses the Block component's size if present.
func (s *PhysicsSystem) InitFromWorld(objectWorld *object.World) {
	for _, obj := range objectWorld.Objects() {
		pb := obj.PhysicsBody()
		if pb == nil || pb.Body != nil {
			continue
		}
		t := obj.Transform()
		if t == nil {
			continue
		}
		// Prefer Block dimensions when present (e.g. static bodies with only body_type in YAML); else use physics_body width/height.
		w, h := pb.Width, pb.Height
		if c := obj.GetComponent("block"); c != nil {
			if blk, ok := c.(*object.Block); ok && blk.Width > 0 && blk.Height > 0 {
				w, h = blk.Width, blk.Height
			}
		}
		if w <= 0 {
			w = 64
		}
		if h <= 0 {
			h = 64
		}
		density := pb.Density
		if pb.Mass > 0 && w > 0 && h > 0 {
			density = pb.Mass / (w * h)
		}
		if density <= 0 {
			density = 1
		}
		// For circle, use radius and derive w,h for sync (2*radius). Prefer Ball component radius when present.
		shape := pb.Shape
		radius := pb.Radius
		if shape == physics.ShapeCircle {
			if c := obj.GetComponent("ball"); c != nil {
				if ball, ok := c.(*object.Ball); ok && ball.Radius > 0 {
					radius = ball.Radius
				}
			}
			if radius > 0 {
				w, h = radius*2, radius*2
			}
		}
		center := physics.Vec2{
			X: t.X + w*0.5 + pb.OffsetX,
			Y: t.Y + h*0.5 + pb.OffsetY,
		}
		body, err := s.world.CreateBody(physics.BodyDef{
			Position:    center,
			Width:       w,
			Height:      h,
			Radius:      radius,
			Type:        pb.BodyType,
			Shape:       shape,
			Density:     density,
			Restitution: pb.Restitution,
			Friction:    pb.Friction,
		})
		if err != nil {
			continue
		}
		pb.Body = body
		pb.Width = w
		pb.Height = h
		if pb.BodyType == physics.BodyStatic {
			s.staticDebug = append(s.staticDebug, staticDebugInfo{name: obj.Name, center: center, width: w, height: h})
		} else {
			s.kinematicBodies[obj.Name] = kinematicEntry{
				body: body, width: w, height: h,
				offsetX: pb.OffsetX, offsetY: pb.OffsetY,
			}
		}
	}
}

// LogBodies prints all physics body positions and sizes in game units (for scale/collision debugging).
func (s *PhysicsSystem) LogBodies() {
	log.Println("--- physics bodies (game units: center X,Y and size WxH) ---")
	for name, entry := range s.kinematicBodies {
		pos := entry.body.GetPosition()
		log.Printf("  [moving] %q center=(%.1f, %.1f) size=%.1fx%.1f topLeft=(%.1f, %.1f)",
			name, pos.X, pos.Y, entry.width, entry.height,
			pos.X-entry.width*0.5, pos.Y-entry.height*0.5)
	}
	for _, d := range s.staticDebug {
		tlX := d.center.X - d.width*0.5
		tlY := d.center.Y - d.height*0.5
		log.Printf("  [static] %q center=(%.1f, %.1f) size=%.1fx%.1f topLeft=(%.1f, %.1f)",
			d.name, d.center.X, d.center.Y, d.width, d.height, tlX, tlY)
	}
	log.Println("--- end physics bodies ---")
}

// DebugSummary returns a short string describing body count and bounds (for on-screen debug).
func (s *PhysicsSystem) DebugSummary() string {
	return fmt.Sprintf("physics: %d kinematic, %d static", len(s.kinematicBodies), len(s.staticDebug))
}

// DrawDebug draws collision box outlines in game coordinates (kinematic/dynamic=green, static=red).
// Moving bodies are drawn at the object's transform (sprite position) so the box aligns with the sprite.
func (s *PhysicsSystem) DrawDebug(screen *ebiten.Image, objectWorld *object.World) {
	const strokeWidth = 2.0
	for name, entry := range s.kinematicBodies {
		var x, y float32
		if objectWorld != nil {
			if obj := objectWorld.Find(name); obj != nil && obj.Transform() != nil {
				t := obj.Transform()
				x = float32(t.X + entry.offsetX)
				y = float32(t.Y + entry.offsetY)
			} else {
				pos := entry.body.GetPosition()
				x = float32(pos.X - entry.width*0.5)
				y = float32(pos.Y - entry.height*0.5)
			}
		} else {
			pos := entry.body.GetPosition()
			x = float32(pos.X - entry.width*0.5)
			y = float32(pos.Y - entry.height*0.5)
		}
		vector.StrokeRect(screen, x, y, float32(entry.width), float32(entry.height), strokeWidth, color.RGBA{0, 255, 0, 200}, false)
	}
	for _, d := range s.staticDebug {
		x := float32(d.center.X - d.width*0.5)
		y := float32(d.center.Y - d.height*0.5)
		vector.StrokeRect(screen, x, y, float32(d.width), float32(d.height), strokeWidth, color.RGBA{255, 0, 0, 200}, false)
	}
}

// Step advances the physics simulation by dt seconds.
func (s *PhysicsSystem) Step(dt float64) {
	s.world.Step(dt)
}

// SyncToWorld copies kinematic body positions (center) to the corresponding GameObjects' Transforms (top-left).
func (s *PhysicsSystem) SyncToWorld(objectWorld *object.World) {
	for name, entry := range s.kinematicBodies {
		obj := objectWorld.Find(name)
		if obj == nil {
			continue
		}
		t := obj.Transform()
		if t == nil {
			continue
		}
		pos := entry.body.GetPosition()
		t.X = pos.X - entry.width*0.5 - entry.offsetX
		t.Y = pos.Y - entry.height*0.5 - entry.offsetY
	}
}
