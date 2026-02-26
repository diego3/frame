# Game Concept: Knight Platformer

## Vision

A small 2D platform game built with the goengine, starring the existing knight character. The player runs, jumps, dashes, and fights through a handful of levels, avoiding hazards and reaching the goal to progress. The focus is on tight controls, clear win/lose conditions, and a polished but minimal scope suitable for proving out the engine and shipping a complete game.

**One-line pitch:** A compact 2D platformer where a knight uses movement and dash to overcome obstacles and hazards and reach the goal across multiple levels.

---

## Core Loop

1. **Start level** – Knight spawns at a checkpoint; level layout, platforms, hazards, and goal are fixed (data-driven from YAML).
2. **Play** – Player moves (run, jump, dash), avoids hazards (pits, enemies or deadly zones), and can collect optional pickups (e.g. coins) for score.
3. **Outcome** – **Win:** reach the goal → advance to next level (or “you win” screen). **Lose:** death (fall in pit, touch hazard) → respawn at checkpoint or restart level, with possible lives.
4. **Meta** – Main menu to start; level select or linear progression; optional settings. Between levels, brief feedback (score, “level complete”) before continuing.

The loop is intentionally minimal: no complex systems, no inventory—just movement, risk, and reaching the end of each level.

---

## Feel & Inspiration

- **Control feel:** Responsive movement (existing run/dash), plus jump for vertical platforming. Combat (attack) can be used sparingly (e.g. break crates, hit switches) or omitted in early scope.
- **Pacing:** Short levels (1–3 minutes each), low punishment (respawn or one-life-per-level). The game should feel “complete” in under 30 minutes for a first playthrough.
- **Visual/Audio:** Parallax background layers for depth; clear read of platforms, hazards, and goal. Background music per level or area; SFX for jump, dash, collect, goal, death. HUD for lives and score so the player always knows their state.
- **Inspiration:** Classic 2D platformers (simple goal: get to the end), with a small moveset (run, jump, dash) and clear rules. No narrative requirement; optional light theme (e.g. knight rescuing something, or just “reach the flag”).

---

## Features (Scope)

### Must-have (MVP)

- **Levels:** Multiple levels (e.g. 3–5), each loaded from data (YAML); scene manager switches between them.
- **Knight as player:** Run, jump, dash; camera follows knight; death on pits/hazards with respawn or level restart.
- **Goal:** A defined goal zone or object; reaching it triggers “level complete” and next level (or victory).
- **Hazards:** At least one type (e.g. pits by Y-death, or static hazard bodies); collision/trigger to detect contact and trigger death.
- **Camera:** Follows the knight so levels can be wider (and optionally taller) than the screen.
- **Background:** At least one background image per level (or global); optional 2–3 layer parallax for depth.
- **Audio:** Background music (looping per level or menu); SFX for key actions (jump, dash, goal, death, collect).
- **HUD:** Displays lives (if used) and score (if collectibles exist); minimal but always visible during play.
- **Menus:** Main menu (start, maybe settings); level complete / game over / victory screens with “continue” or “retry.”

### Should-have (Polish)

- **Collectibles:** Coins or similar; add to score; optional “collect all” for extra feedback.
- **Lives:** Optional lives system; on death, lose one life and respawn; game over when lives = 0.
- **Parallax:** Multiple background layers with different scroll factors for a sense of depth.
- **Level select:** After first run or from main menu, choose which level to play (unlock progression optional).

### Could-have (Later)

- **Combat:** Use existing attack for obstacles or simple enemies.
- **Checkpoints:** Respawn at last checkpoint instead of level start on death.
- **Settings:** Volume (BGM/SFX), fullscreen, controls (if rebindable later).
- **More levels:** Expand to 8–10 levels once core loop is solid.

---

## Target

- **Platform:** Desktop (current engine target).
- **Audience:** Solo developer / portfolio piece; optionally “small game for friends or itch.io.” No requirement for mobile or console in this concept.
- **Tech:** goengine (Ebiten, data-driven scenes, physics, scene manager). Engine work aligned with this game: camera, collision/triggers, jump, background/parallax, BGM, HUD.

---

## Summary

The game is a **minimal 2D platformer** built to validate and showcase the engine: the knight runs, jumps, and dashes through a few levels, avoids hazards, reaches the goal, and optionally collects items for score. Scope stays small so that “game complete” (menus, levels, win/lose, HUD, audio, background) is achievable while the engine gains the core systems (camera, triggers, BGM, HUD, parallax) needed for this and future projects.
