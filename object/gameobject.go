package object

import "sync/atomic"

var nextID uint64

// GameObject is a container for components, similar to Unity's GameObject.
type GameObject struct {
	ID         uint64
	Name       string
	Active     bool
	components map[string]Component
}

// NewGameObject returns a new active GameObject with the given name.
func NewGameObject(name string) *GameObject {
	id := atomic.AddUint64(&nextID, 1)
	return &GameObject{
		ID:         id,
		Name:       name,
		Active:     true,
		components: make(map[string]Component),
	}
}

// AddComponent attaches a component. Uses c.Type() as the key; duplicate types replace the previous.
func (g *GameObject) AddComponent(c Component) {
	if c == nil {
		return
	}
	g.components[c.Type()] = c
}

// GetComponent returns the component with the given type name, or nil.
func (g *GameObject) GetComponent(typeName string) Component {
	return g.components[typeName]
}

// Transform returns the Transform component if present.
func (g *GameObject) Transform() *Transform {
	if t, ok := g.components["transform"].(*Transform); ok {
		return t
	}
	return nil
}

// Animator returns the Animator component if present.
func (g *GameObject) Animator() *Animator {
	if a, ok := g.components["animator"].(*Animator); ok {
		return a
	}
	return nil
}

// ResetAnimation restarts the named spritesheet from frame 0 (for one-shot animations). No-op if not found.
func (g *GameObject) ResetAnimation(name string) {
	key := "spritesheet:" + name
	if c := g.components[key]; c != nil {
		if s, ok := c.(*Spritesheet); ok {
			s.Reset()
		}
	}
}

// PhysicsBody returns the PhysicsBody component if present.
func (g *GameObject) PhysicsBody() *PhysicsBody {
	if pb, ok := g.components["physics_body"].(*PhysicsBody); ok {
		return pb
	}
	return nil
}
