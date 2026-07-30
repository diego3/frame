package object

import "sync/atomic"

var nextID uint64

// GameObject is a container for components, similar to Unity's GameObject.
type GameObject struct {
	ID          uint64
	Name        string
	Active      bool
	IsPrototype bool // true for a template GameObject never run/drawn, only cloned (see Clone)
	components  map[string]Component
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

// Clone returns a new, independent GameObject named name with a fresh ID, Active set true, and a
// Clone() of every component g has (see the Prototype pattern: a template GameObject marked
// IsPrototype is never itself run or drawn — see Manager.Update/Draw — only cloned to spawn real
// instances, e.g. games/metalslug_demo's "projectile_prototype"). The clone's IsPrototype is
// always false, regardless of g's.
func (g *GameObject) Clone(name string) *GameObject {
	clone := NewGameObject(name)
	for typeName, c := range g.components {
		clone.components[typeName] = c.Clone()
	}
	return clone
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

// AnimatedComponent returns the single component selected by the Animator, when one is present with a
// non-empty Current (e.g. the "spritesheet:run" component while Current is "run"). ok is false when there
// is no Animator constraint, meaning callers should consider all components instead of just one.
// Centralizing this here keeps Animator's "only one active animation" rule out of Manager.Update/Draw.
func (g *GameObject) AnimatedComponent() (c Component, ok bool) {
	anim := g.Animator()
	if anim == nil || anim.Current == "" {
		return nil, false
	}
	return g.components["spritesheet:"+anim.Current], true
}
