# ADR-007: Game object pool pattern

## Status

Proposed.

## Context

Games often create and destroy many objects every frame (bullets, particles, enemies). Allocating with `new` or `make` and letting the garbage collector clean up can cause:

- **GC spikes** – the collector pauses the game when it runs.
- **Frame hitches** – stutter when many objects are created or freed at once.

We need a way to reuse objects instead of allocating new ones every time.

## Decision

Use an **object pool**: pre-allocate a set of objects and reuse them.

1. **When you need an object** – take one from the pool (or create one only if the pool is empty).
2. **When you're done with it** – reset its state and put it back in the pool instead of discarding it.

Apply this to high-volume, short-lived objects (e.g. projectiles, particles, one-shot effects). Do **not** pool everything; use it where profiling shows allocation pressure.

## Tradeoffs

| | Benefit | Cost |
|---|--------|------|
| **Performance** | Fewer allocations → less GC work → smoother frame rate. | You must reset state when returning to the pool; forgetting can cause subtle bugs (e.g. a "dead" bullet still drawn). |
| **Complexity** | Hot paths (Update/Draw) stay predictable. | More code: pool type, Get/Return (or Put) API, and clear reset rules. |
| **Memory** | No allocation churn in gameplay. | Pool holds memory for max capacity even when not all objects are in use. |

**When to use:** Many instances of the same type created/destroyed often (e.g. bullets, particles).  
**When to skip:** Long-lived or few objects; prefer normal allocation until profiling shows a problem (see ADR-006: measure first).

## Summary

- **Do:** Pool high-volume, short-lived game objects; document reset behavior.
- **Don’t:** Pool by default; add pools only when allocation shows up in profiles.

## References

- ADR-006: Coding standards (frame budget, minimize allocations in Update/Draw, measure before optimizing).
- Common pattern in game engines (e.g. Unity `Object.Instantiate` vs pooling; Ebiten/Go: reuse structs/slices).
