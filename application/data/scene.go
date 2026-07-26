package data

import (
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"

	"goengine/object"
	"goengine/ports"
)

// SceneDef is the data-driven definition of a scene (list of game objects).
type SceneDef struct {
	Objects []ObjectDef `yaml:"objects"`
}

// ObjectDef defines one game object and its components.
type ObjectDef struct {
	Name       string                   `yaml:"name"`
	Active     *bool                    `yaml:"active"`    // nil = true
	Prototype  bool                     `yaml:"prototype"` // true = template only, never run/drawn; see object.GameObject.Clone
	Components []map[string]interface{} `yaml:"components"`
}

// ComponentBuilder creates a Component from YAML params and the asset loader.
type ComponentBuilder func(params map[string]interface{}, loader ports.AssetLoader) (object.Component, error)

var builders = map[string]ComponentBuilder{}

// RegisterComponentBuilder registers a builder for a component type (e.g. "transform", "sprite", "spritesheet").
func RegisterComponentBuilder(typeName string, fn ComponentBuilder) {
	builders[typeName] = fn
}

// LoadScene loads a scene definition from a YAML file on the OS filesystem.
func LoadScene(path string) (*SceneDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load scene %q: %w", path, err)
	}
	var def SceneDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse scene %q: %w", path, err)
	}
	return &def, nil
}

// LoadSceneFS loads a scene definition from fsys at path. Use for embedded assets (e.g. WASM builds).
func LoadSceneFS(fsys fs.FS, path string) (*SceneDef, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("load scene %q: %w", path, err)
	}
	var def SceneDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse scene %q: %w", path, err)
	}
	return &def, nil
}

// BuildWorld creates an object.Manager from a scene definition, using the loader for images.
func BuildWorld(def *SceneDef, loader ports.AssetLoader) (*object.Manager, error) {
	if def == nil {
		return object.NewManager(), nil
	}
	world := object.NewManager()
	for _, o := range def.Objects {
		obj := object.NewGameObject(o.Name)
		if o.Active != nil && !*o.Active {
			obj.Active = false
		}
		obj.IsPrototype = o.Prototype
		for i, c := range o.Components {
			if c == nil {
				continue
			}
			typeVal, _ := c["type"].(string)
			if typeVal == "" {
				return nil, fmt.Errorf("object %q component %d: missing type", o.Name, i)
			}
			fn, ok := builders[typeVal]
			if !ok {
				return nil, fmt.Errorf("object %q component %d: unknown type %q", o.Name, i, typeVal)
			}
			comp, err := fn(c, loader)
			if err != nil {
				return nil, fmt.Errorf("object %q component %d (%s): %w", o.Name, i, typeVal, err)
			}
			obj.AddComponent(comp)
		}
		world.Add(obj)
	}
	return world, nil
}
