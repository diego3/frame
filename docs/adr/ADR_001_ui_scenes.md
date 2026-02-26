# ADR-001: UI and scene data model

## Status

In Review.

## Context

We need a clear data model for:

- **Scene management**: switching between screens (splash, main menu, settings, levels, pause, loading).
- **Data-driven UI**: describing screens and their elements (buttons, labels, panels) in YAML instead of hardcoding in Go.
- **Optional hierarchy**: parent/child relationships for UI elements (e.g. panels containing buttons), similar to design tools and other engines, without adding undue complexity.

Open questions were:

- Whether to introduce a **SceneNode** (a unified node type for screen, UI, gameplay, etc.).
- How to type or separate **screen**, **UI**, and **gameplay**.
- Whether and how to support **parent/child** for screen elements.

## Decision

### 1. SceneManager and Scene (unchanged)

- **SceneManager**: Keeps responsibility to register scenes by id and switch the current one. It does not own or understand node trees.
- **Scene**: Remains the unit of switching (one scene = one thing that gets Setup/Update/Draw). It owns its gameplay (e.g. object world) and its UI (flat list today; tree later if needed).

### 2. No generic SceneNode for now

- We do **not** introduce a single **SceneNode** type that mixes screen, UI, and gameplay in one tree.
- **Rationale**: Different concerns need different data (e.g. UI has layout/visibility, gameplay is the existing object world). Keeping separate trees per concern is simpler and avoids a large union type or heavy abstraction. A unified node graph can be added later if an editor or single serialization format requires it.

### 3. Screen as a concept, not a node type

- **Screen** means “what is on display for this scene.” For now it is not a separate entity in the data model: the current scene *is* the screen.
- Optionally, a **Screen** descriptor in config/YAML can list `ui_file` and `gameplay_scene` for that screen; the scene loads that UI and world. Screen is then a descriptor, not a node in a graph.

### 4. Typing: screen vs UI vs gameplay

- **Screen**: Which full-screen layout is shown (often 1:1 with current scene).
- **UI**: Interactive overlay (buttons, labels, panels). Can be modeled as its own tree (see below).
- **Gameplay**: Existing object world (knights, physics, etc.). Stays separate from the UI tree.
- “Type” is thus “which subsystem” (screen / UI / gameplay), not a tag on every node in a single graph.

### 5. Parent/child only for UI, and only when needed

- **Scope**: Parent/child is introduced only for **UI** (layout, visibility, draw order), not for a global scene graph.
- **Minimal design**:
  - One tree per scene’s UI: root (e.g. “screen” or “root”) and children (panels, groups, leaves like buttons/labels).
  - Each node: local position (and size where relevant); optional parent; children list. World position computed by walking up to root (or cached when parent moves).
  - No layout engine in the first version: no anchors or flex; only “children have local offset from parent” for grouping and moving groups.
- **YAML**: Either flat list with `parent_id` (easy to parse, build tree in memory) or recursive `children:` (readable for small trees). Choose when implementing.
- **When to add**: Add UI hierarchy when there is a concrete need (e.g. “main menu has a background panel with buttons inside”). Until then, a flat list of elements under a single root is enough.

### 6. Minimal data model summary


| Concept      | Role                                                                                                                         |
| ------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| SceneManager | Registry of scene ids → factory; current scene; `SwitchTo(id)`.                                                              |
| Scene        | Unit of switching; owns gameplay world and UI (flat list or UI tree).                                                        |
| Screen       | Descriptor (optional): e.g. `ui_file`, `gameplay_scene`; or simply “current scene’s content.”                                |
| SceneNode    | Not introduced. Defer until a single graph is clearly needed.                                                                |
| UI hierarchy | Optional: UINode (or UIElement) with type (panel, button, label), local position, parent, children. One tree per scene’s UI. |


## Consequences

### Positive

- Clear separation of concerns: scene management, UI, and gameplay stay distinct.
- Data-driven UI can be added with a flat list of elements (and optional hierarchy later) without touching the scene manager.
- Parent/child is scoped to UI only, keeping the change set and cognitive load small.
- We avoid a generic SceneNode type and optional payloads until we have a concrete need.

### Negative

- No single “scene graph” to inspect or serialize; multiple structures (scene registry, UI tree, object world) coexist.
- If we later want an editor or one unified format, we may introduce a SceneNode (or similar) and map it onto these structures.

## References

- Internal discussion: UI/scenes data modeling (SceneManager, SceneNode tradeoffs, screen/UI/gameplay typing, parent/child for UI).
- Existing code: `scene/manager.go`, `ports.Scene`, `ui.Container`, `game.Game` (scene manager + UIRoot).

