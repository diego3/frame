# ADR-005: Scripting language integration (Lua) with Go

## Status

Accepted. **Option 1 (Pure Go VM) implemented.**

## Context

We want to add a **scripting language** so that part of the game’s behavior can be written outside of Go and changed without recompiling the engine. Typical uses:

- **Game logic in scripts:** Level-specific behavior (e.g. “when player touches this object, open the door”), win/lose conditions, simple AI.
- **Designers and modders:** People who are not Go developers can change or extend behavior by editing script files (e.g. `.lua`).
- **Data-driven behavior:** YAML (or similar) can reference script names or snippets; the engine runs them at the right time.

**Lua** is a common choice in games: small, fast, easy to embed, and many developers already know it. This ADR compares two practical ways to **integrate Lua with our Go engine** so we can run Lua scripts from Go and call back into Go from Lua (e.g. “get player position,” “play sound,” “switch scene”).

---

## Problem

We need to:

1. **Run Lua code** from Go (e.g. when a level loads or when the player touches a trigger).
2. **Call Go from Lua** so scripts can use engine features (get/set entity data, play sounds, emit events, change scene).
3. **Keep the solution maintainable** and, if possible, avoid build or portability issues (e.g. CGo can complicate cross-compilation).

---

## Options

We consider **two approaches** to integrate Go with Lua. Both allow running Lua scripts and exposing Go functions to Lua; they differ in how Lua is implemented and what we depend on.

---

### Option 1: Pure Go Lua VM (e.g. gopher-lua)

**What it is:** A Lua 5.1 implementation written entirely in Go. No C code, no CGo. The script runs inside our Go process; we create a “state,” run Lua code, and register Go functions so Lua can call them.

**How it works (simplified):**

1. We add a dependency (e.g. `github.com/yuin/gopher-lua`).
2. We create an `LState` (Lua state) when we need to run scripts (e.g. per level or one global).
3. We register Go functions with the state (e.g. `engine.play_sound(path)`), so from Lua we can call `engine.play_sound("footstep.ogg")`.
4. We run a script with `state.DoFile("level1.lua")` or `state.DoString("...")`. The script can call those Go functions and we can call Lua functions from Go.

**Pros:**

- **No CGo:** Pure Go. Builds everywhere Go builds; cross-compilation (e.g. for other OS) stays simple.
- **Easy to reason about:** Script runs in the same process; calling Go from Lua is just registering functions. Good for juniors: “Lua calls these Go functions we give it.”
- **Single dependency:** One library; no system Lua or C compiler required.
- **Control:** We control when scripts run and when the state is created/destroyed (e.g. one state per level, clear on scene change).

**Cons:**

- **Lua 5.1 only:** Not the latest Lua (5.4). For a small game and typical scripting needs, 5.1 is usually enough.
- **Not the “official” C Lua:** If we ever need 100% compatibility with existing C Lua extensions or exact Lua 5.2/5.3/5.4 behavior, we’d need a different approach.
- **Performance:** For heavy script use (thousands of calls per frame), a C Lua VM can be faster; for normal game logic (triggers, a few calls per frame), the pure Go VM is typically fine.

**Summary:** Best fit when we want **no CGo**, **simple builds**, and **good enough** Lua for game logic and triggers. Easiest to adopt for a junior: add one package, register Go functions, run scripts.

---

### Option 2: CGo bindings to the C Lua library

**What it is:** We use the real Lua implementation (written in C) by linking it into our Go program via CGo. A Go wrapper library (e.g. `github.com/aarlin/go-lua` or similar) exposes the C Lua API to Go so we can run Lua and exchange data between Go and Lua.

**How it works (simplified):**

1. We need a C compiler and the Lua C library (or a bundled copy) to build the project.
2. We add a Go package that uses CGo to call the C Lua API (create state, run file, push/pull values, register C/Go functions).
3. We create a Lua state, register Go functions (exposed via C) so Lua can call them, and run scripts the same way conceptually as in Option 1.
4. Scripts run in the same process; the difference is that the Lua VM is the standard C implementation.

**Pros:**

- **Standard Lua:** Full compatibility with Lua 5.x (whichever version we link). Any Lua documentation or C Lua extension applies.
- **Performance:** The C Lua VM is highly optimized; can matter if we run a lot of script code every frame.
- **Ecosystem:** If we ever want to use existing C Lua libraries (e.g. a specific Lua module), we can link them.

**Cons:**

- **CGo required:** We need a C toolchain to build. Cross-compiling from one OS to another (e.g. Windows → Linux) is harder and sometimes requires extra setup. Not ideal for “one `go build` everywhere.”
- **Build and portability:** New contributors (or CI) must have Lua (or bundled Lua) and C compiler. Juniors may hit “CGo disabled” or “Lua not found” issues.
- **More moving parts:** Two runtimes (Go and C Lua), wrapper code, and possible memory/lifecycle details (who frees what). Slightly harder to debug and explain.

**Summary:** Best fit when we **must** have standard Lua or maximum script performance and we **accept** CGo and build complexity. Less friendly for juniors and portability.

---

## Recommendation

For a **small platform game** and a team that values **simple builds and readability**:

- Prefer **Option 1 (Pure Go Lua VM, e.g. gopher-lua)** unless we have a concrete need for C Lua (e.g. a specific C Lua library or Lua 5.4 features).
- Revisit **Option 2** only if we later need full Lua compatibility or prove that script performance is a bottleneck.

---

## Consequences

### If we choose Option 1 (Pure Go VM)

- **Positive:** Simple integration; no CGo; easy to document (“we use gopher-lua; here’s how we register Go functions and run scripts”). Juniors can add new script-callable functions in Go and write Lua without dealing with C.
- **Negative:** We are tied to Lua 5.1 and that VM’s behavior; if we later need Lua 5.4 or a C Lua extension, we’d need to evaluate Option 2 or another binding.

### If we choose Option 2 (CGo + C Lua)

- **Positive:** Standard Lua and best performance; future-proof for heavy or exotic script use.
- **Negative:** Build and portability get more complex; onboarding and “just run the game” become harder. We take on CGo and C Lua lifecycle.

---

## References

- **gopher-lua (Option 1):** [github.com/yuin/gopher-lua](https://github.com/yuin/gopher-lua) – Pure Go Lua 5.1 VM.
- **Lua in C:** [lua.org](https://www.lua.org/) – Official Lua; C implementation.
- **CGo:** [Go docs – CGo](https://pkg.go.dev/cmd/cgo) – Build constraints and portability implications.

## Implementation (Option 1)

- **Package `script`:** VM wraps `gopher-lua` LState; `NewVM()`, `RegisterEngine(name, fns)`, `DoFile(path)`, `DoString(s)`, `CallFunc(name, args...)`, `Close()`. `EngineFuncs(playSound, switchScene, quit)` returns the map for `engine.play_sound`, `engine.switch_scene`, `engine.quit`.
- **Game:** Creates VM in `Init()`, registers engine callbacks (play sound via loader, switch scene/quit via event bus), closes VM in `Shutdown()`. `ScriptVM()` returns the VM for running scripts (e.g. from a scene or level loader).
- **Sample script:** `scripts/sample.lua` shows engine API usage and a function callable from Go via `CallFunc`.
