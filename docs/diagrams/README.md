# Sequence Diagrams

Didactic sequence diagrams for a new dev learning how this codebase's main flows actually work,
from process boot down to a specific gameplay action — and how the Application, engine (Logic),
and View layers (ADR-003) talk to each other along the way. Every diagram is grounded in the real
call chain in the code as of when it was written, not a simplified/aspirational version — where
something is a nuance worth knowing (e.g. `on_event` broadcasting to *every* loaded script, or
re-entrant `emit` calls during script updates), there's a note callout for it.

## Reading order

| # | Diagram | What it teaches |
|---|---|---|
| 1 | [`01_engine_boot.puml`](01_engine_boot.puml) | `main.go` → `engine.New` → `Game.Init` → `Scene.Setup` → YAML scene loading and Box2D body creation. The one-time bootstrap. |
| 2 | [`02_frame_update.puml`](02_frame_update.puml) | The core game loop's `Update` half: input → intent events → the event bus → script `on_event`/`update` calls → physics step → contact events → camera follow. Read this one closely; almost everything else is a special case of it. |
| 3 | [`03_frame_draw.puml`](03_frame_draw.puml) | The `Draw` half: the world-buffer + camera-translate trick, and where Drawer components fit in. |
| 4 | [`04_player_shoot.puml`](04_player_shoot.puml) | A concrete vertical slice through #2: pressing a key ends up cloning a `projectile_prototype` GameObject, via a script that only *decides*, never builds objects itself. |
| 5 | [`05_enemy_bomber_camera_shake.puml`](05_enemy_bomber_camera_shake.puml) | The richest example: an enemy's own cooldown, a shared script's per-instance `Timer` component, the Prototype-spawn mechanism, and the (new) `process.Manager`-based camera shake reacting to an explosion — all in one flow. |
| 6 | [`06_physics_contact_and_jump.puml`](06_physics_contact_and_jump.puml) | Box2D `BeginContact`/`EndContact` events driving a script's own state (`is_grounded`), and why that state is a dict, not a bool. |

Diagrams 1–3 describe the generic engine; 4–6 are concrete `games/metalslug_demo` examples chosen
to each exercise a different part of the architecture (scripting conventions, the Prototype
pattern, the process manager, physics contacts) rather than to be exhaustive.

## Rendering

Each `.puml` has a matching `.png` committed alongside it, so the diagram renders straight on
GitHub/GitLab without anyone needing PlantUML installed just to *read* one. The `.puml` is still
the source of truth — regenerate the `.png` from it whenever the `.puml` changes, don't hand-edit
the image. Any of these work:

- **VS Code**: the "PlantUML" extension (jebbs.plantuml) renders a live preview and can export.
- **CLI**, if you have a local PlantUML jar/package and Graphviz installed:
  ```bash
  plantuml -tpng docs/diagrams/*.puml   # regenerates every .png in place
  ```
- **No local install**: paste a file's contents into the [PlantUML online server](https://www.plantuml.com/plantuml/uml/).
  Only do this for these engine-architecture diagrams — they contain no project-specific secrets,
  but treat that server as public regardless.

## Keeping these accurate

These will drift as the code changes — they are not generated from source. When a change touches
one of these flows (a new event type, a new step in `Scene.Update`, a changed script API), update
the relevant `.puml` **and regenerate its `.png`** in the same PR, the same way ADRs get updated
for architectural decisions. A stale `.png` next to a changed `.puml` is worse than no image at
all — it looks authoritative but silently lies.
