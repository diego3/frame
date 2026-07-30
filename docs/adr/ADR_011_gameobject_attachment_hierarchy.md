# ADR-011: GameObject attachment hierarchy (parent/child transforms)

## Status

Proposed.

## Context

We want a GameObject to be attachable to another GameObject at runtime so it follows the
parent's position/rotation automatically — e.g. a sword `GameObject` attached to (and later
detached from) the player, or a bomb visually stuck to whatever launched it. This is the classic
"scene graph" capability: a transform hierarchy where a child's world position is derived from
its parent's, not tracked independently.

### This was already considered once, and scoped away from gameplay

ADR-001 ("UI and scene data model") looked at exactly this question and deliberately said no,
for gameplay:

> "Scope: Parent/child is introduced only for UI (layout, visibility, draw order), not for a
> global scene graph." ... "When to add: Add UI hierarchy when there is a concrete need... Until
> then, a flat list of elements under a single root is enough."

That "concrete need" is what this ADR is for. It does **not** revisit ADR-001's UI decision —
UI's flat-list-for-now status is unrelated and unaffected — it only addresses the gameplay side
ADR-001 explicitly deferred. Worth noting: even UI's optional parent/child was never built
(`view/ui.Container` is still a flat `[]Element`), so there is no existing hierarchy
implementation anywhere in the engine to build on or reuse.

### Current state

`object.Transform` (`object/transform.go`) is a flat, world-space-only struct:

```go
type Transform struct {
	X, Y   float64 // position in logical coordinates
	Angle  float64 // rotation in radians
	ScaleX float64
	ScaleY float64
}
```

No parent reference exists anywhere — not on `Transform`, not on `GameObject`. `object.Manager`
holds objects in a flat slice (`Objects()`) and updates/draws them independently. The only
precedent for something *external* overwriting a `Transform` each frame is
`PhysicsSystem.SyncToWorld`, which copies a Box2D body's actual position back onto its
`GameObject`'s `Transform` after each physics step — a useful shape to reuse here, not a
hierarchy itself.

### Comparing against Game Coding Complete

*Game Coding Complete* (Ch. 9–10) solves this with a **scene graph**: a tree of `SceneNode`s,
separate from the actor/GameObject system, where each node composes its world transform from its
parent's (`ToWorld` = `parent.ToWorld * local`, via a 4×4 matrix stack, since the book targets
Direct3D). Actors don't own their visuals directly — a render component attaches an actor to a
node in the graph. Attaching a weapon to a character's hand is done by parenting the weapon's
node under the hand/bone node; the weapon's world transform is then recomputed automatically
every frame with zero code in the weapon itself.

The book is 3D/DirectX-shaped, but the underlying idea is dimension-agnostic — the child's world
transform is always "parent's world transform, then apply my local offset," it's only the math
that differs. For this engine's 2D case, the composition is: rotate the local offset by the
parent's `Angle` (`vec2.Vector.Rotate`, already implemented), scale it by the parent's
`ScaleX/ScaleY`, then translate by the parent's `X, Y`; the child's own `Angle` and
`ScaleX/ScaleY` add on top of the parent's. No matrix type is needed — this stays within what
`vec2.Vector` already provides, consistent with why that package exists (see its own doc comment
and the earlier vec2-vs-`go-gl/mathgl` comparison referenced there).

### Constraints

- **Backward compatible by default.** The overwhelming majority of `GameObject`s have no parent
  and must keep `Transform.X/Y` meaning "world position," unchanged, with zero added cost for
  them.
- **Composes with Prototype spawning (ADR-007's sibling pattern).** Equipping a sword should be
  "clone `sword_prototype`, then attach the clone to the player" — the same `spawnEntity`
  mechanism already used for projectiles/hazards, not a separate spawning path.
  `GameObject.Clone()` currently only clones a single object's own components; whether cloning a
  parent should also deep-clone its attached children is an open question below, not yet decided.
- **Must not fight `PhysicsSystem.SyncToWorld`.** A physics-driven parent (the player) has its
  `Transform` overwritten from its Box2D body every frame; attachment resolution must run
  *after* that, and an attached child should generally not carry its own `PhysicsBody` — a sword
  isn't independently simulated, it rides its parent.
- **No globals, minimize per-frame allocation** (ADR-006, CLAUDE.md) — resolution must be a
  plain method call over existing slices, not a new global registry.

## Approach 1: No engine feature — handle it per-script

The wielding entity's own script (e.g. `player_controller.py`) copies its `Transform.X/Y` (plus
a fixed offset for hand position, adjusted by facing) onto the sword's `Transform` every
`update(dt)`, using `engine.get_entity_position`/an equivalent setter.

| Pros | Cons |
|------|------|
| Zero engine changes; ships today with what already exists (`engine.get_entity_position` is already used this way in `enemy_bomber.py` for aiming). | Every attachable item needs its own bespoke follow code, duplicated per script/language (Lua vs. Python, per CLAUDE.md's Python-only-going-forward rule — still two implementations exist for existing scripts). |
| No new abstraction to design or maintain. | No generic "attach/detach at runtime" — equipping/unequipping means the script conditionally running that copy logic, easy to get subtly wrong per item. |
| Fine for exactly one attachment relationship in one demo. | Doesn't scale past a handful of cases; this is exactly the kind of hand-rolled timer/position logic the `game-architecture` skill tries to steer people away from. |

## Approach 2: Revive ADR-001's rejected generic SceneNode, now covering gameplay too

Build the unified node tree ADR-001 declined (one graph type for UI *and* gameplay), and make
every `GameObject` a node in it.

| Pros | Cons |
|------|------|
| One hierarchy concept for the whole engine instead of two. | Directly reopens ADR-001's decision, whose rationale (different concerns need different data — UI has layout/visibility, gameplay has physics/scripts/components) still holds. A union type big enough for both ends up implementing neither well. |
| Would also finally deliver UI's still-unbuilt optional hierarchy. | Large, high-risk refactor of `object.Manager` and `view/ui` at once, for a need that's currently scoped to gameplay only. |
| | UI has no concrete need for hierarchy yet either (ADR-001 already said so, still true) — bundling it in means shipping speculative UI work to get gameplay attachment. |

## Approach 3: Optional parent + local offset on GameObject, resolved by a small AttachmentSystem (recommended)

Add an optional parent reference and a local offset to `GameObject`, and a small system that
resolves world `Transform`s once per frame — the gameplay-only, minimal-surface-area version of
a scene graph, deliberately *not* touching UI.

```go
// object/gameobject.go
type GameObject struct {
	// ... existing fields
	ParentID    uint64    // 0 = no parent
	LocalOffset Transform // position/angle/scale relative to ParentID's Transform
}
```

```go
// New: view/scene/attachment.go (or object/attachment.go)
// Resolve walks every GameObject with a non-zero ParentID and overwrites its Transform from
// parent.Transform ⊕ LocalOffset. Parents must be resolved before their children; since chains
// are expected to be shallow (an item on a hand, not deep skeletal rigs), a single pass with
// parents processed first (or a depth cap) avoids needing a general topological sort.
func Resolve(world *object.Manager) { ... }
```

Called from `Scene.Update`, in this order: `PhysicsSystem.Step` → `PhysicsSystem.SyncToWorld` →
`attachment.Resolve` → `world.Update`. This mirrors the existing precedent of an external system
overwriting `Transform` after physics (`SyncToWorld`), just for a different source of truth.

| Pros | Cons |
|------|------|
| Small, additive, opt-in — objects with `ParentID == 0` (the vast majority) are entirely unaffected, in behavior and in cost. | Yet another thing that can overwrite `Transform` in a given frame (physics, attachment, scripts) — ordering must be documented clearly (this ADR proposes physics → attachment → scripts... see open question below) or bugs get subtle fast. |
| Reuses `vec2.Vector.Rotate` — no new math dependency. | Still need to decide draw order, physics interaction, and `Clone()` semantics explicitly (see below) — this approach doesn't answer those for free. |
| Composes naturally with Prototype spawning: attach = `spawnEntity` a prototype clone, then set `ParentID`. | A very deep or cyclic parent chain is a real footgun; needs an explicit cycle guard even though normal use (item-on-character) is shallow. |
| Matches this codebase's general bias toward small, targeted additions over generic frameworks (TDR-007's same philosophy, applied here to `GameObject` itself). | |

## Approach 4: Physics joints (Box2D weld/revolute) instead of a transform hierarchy

Use `physics/box2d` joints to rigidly (or semi-rigidly) attach two physics bodies, letting Box2D
itself keep them together.

| Pros | Cons |
|------|------|
| Physically correct for cases that should actually simulate (a swinging lantern, a breakable weld, ragdoll limbs). | Wrong tool for the common case (a sword glued to a hand): forces both objects to have a `PhysicsBody` when the child usually shouldn't (see Constraints above), and pays simulation cost for something that's purely visual. |
| Box2D already handles the composition math; no new engine code for the transform side. | Doesn't cover *non-physics* attachment at all (a UI icon following a world object, e.g. a floating name tag) — Approach 3's `Transform`-level solution does. |

Not a replacement for Approach 3 — a genuinely useful *addition* later, for the subset of
attachments that should be physically simulated rather than purely kinematic-follow.

## Comparison at a glance

| | Approach 1 (per-script) | Approach 2 (unified SceneNode) | Approach 3 (Parent+offset, recommended) | Approach 4 (physics joints) |
|--|--|--|--|--|
| Engine changes | None | Large (object.Manager + view/ui) | Small, additive | Small, `physics/box2d` only |
| Reopens ADR-001 | No | Yes | No | No |
| Backward compatible | Trivially (no change) | Requires migrating existing code | Yes, opt-in per object | Yes, opt-in per object |
| Handles non-physics attachment | Yes (manually) | Yes | Yes | No |
| Generic attach/detach API | No | Yes | Yes | Partial (joint create/destroy) |
| Effort | Low | High | Medium | Medium (only for physical cases) |

## Recommendation

Adopt **Approach 3**: an optional `ParentID` + `LocalOffset` on `GameObject`, resolved by a
small `attachment.Resolve` pass run after physics sync and before script/world update. This is
the direct 2D analog of Game Coding Complete's `SceneNode` parenting, scoped to gameplay only —
it does not reopen or require resolving ADR-001's still-separate (and still unbuilt) UI
hierarchy question. **Approach 1** remains a reasonable stopgap if only a single one-off
attachment is ever needed and this ADR isn't implemented yet. **Approach 4** (physics joints) is
worth adding later, additively, for the subset of attachments that should be physically
simulated rather than purely kinematic — it is not in tension with Approach 3.

## Open questions to resolve at implementation time

- **Draw order.** Should an attached child draw using the parent's `Sprite.Layer` plus a delta,
  or fully independently? A sword should draw in front of or behind the hand depending on facing
  — likely needs a small convention, not solved by this ADR.
- **Detach/re-parent API.** Runtime equip/unequip needs a clear entry point (a new
  `AttachRequested`/`DetachRequested` event pair in `events/events.go`, following the existing
  intent-event convention, is the likely shape) rather than scripts poking `ParentID` directly.
- **`Clone()` semantics.** Does cloning a prototype with attached children deep-clone the whole
  subtree, or only the top object? A fully-equipped enemy prototype (with a weapon already
  attached) is a realistic use case that needs an answer.
- **Cycle/depth guard.** `attachment.Resolve` must reject or ignore a parent cycle rather than
  looping forever; a shallow max-depth check is likely enough given expected use (items on
  characters, not deep rigs).

## Consequences

### Positive

- Delivers real, requested gameplay capability (runtime-attachable items) using a pattern with
  direct precedent in *Game Coding Complete* and in this engine's own `SyncToWorld` shape.
- Fully additive: existing GameObjects, scenes, and scripts are unaffected until something
  opts in by setting `ParentID`.
- Composes with the Prototype/`spawnEntity` pattern already established for spawning, instead of
  introducing a second spawning mechanism.

### Negative

- Introduces a third source of per-frame `Transform` mutation (physics, attachment, scripts),
  which needs the update-order rule documented (this ADR proposes one) and respected by future
  contributors.
- Several real design questions (draw order, detach API, `Clone()` of a subtree, cycle guard)
  are intentionally left open here rather than pre-designed, and need a decision before or
  during implementation.

## References

- ADR-001: UI and scene data model — the prior decision this ADR narrows and revisits only for
  gameplay, not UI.
- ADR-003: Layer separation and event flow — any new attach/detach intent event follows its
  existing conventions.
- ADR-006: Coding standards — no globals, minimize per-frame allocation.
- ADR-007: Object pool pattern — attached items are natural Prototype/clone candidates, the same
  spawning mechanism this ADR builds on.
- TDR-007: GameObject type-specific getters — same philosophy applied here (small, targeted
  additions to `GameObject` over a generic framework).
- `vec2` package (`vec2/vec2.go`) — provides the rotation math the 2D transform composition
  needs, with no new dependency.
- `view/scene/physics_system.go`'s `SyncToWorld` — existing precedent for an external system
  overwriting `Transform` after another system's `Update`.
- *Game Coding Complete, 4th Edition* (McShaffry & Graham), Ch. 9–10: Scene graphs and
  hierarchical `SceneNode` transforms — the source pattern this ADR adapts from 3D to 2D.
