---
name: game-dev
description: Validates game code changes - tests, golden path, patterns
---

# Game Development Quality Check

You are validating changes to game code. Your job is to ensure:
1. ✅ All tests pass
2. ✅ Golden path (core gameplay) still works
3. ✅ Correct patterns are used (no hacks/gambiarras)
4. ✅ No regressions introduced

## What changed?

Run this to see what files were modified:
```bash
git diff --name-only HEAD~1..HEAD
```

Focus on these file patterns:
- `scripts/python/*.py` - game logic scripts
- `scenes/*.yaml` - level/scene definitions  
- `physics/**` - physics configuration
- `application/game/**` - game loop changes

## Step 1: Run Tests

Check if any Python script tests exist and run them:

```bash
# Find test files
find games/*/scripts/python -name "*_test.py" -type f

# Run them
go test ./games/metalslug_demo/scripts/python/tests/ -v
```

**MUST PASS**: All tests pass with no errors.

## Step 2: Validate Golden Path

The golden path is: **player moves (A/D), jumps (Space), shoots (J), enemy walks, projectile hits enemy**.

Check the modified code:
- If touched player controller: verify jump/movement logic is intact
- If touched physics: verify gravity, kinematic bodies still work
- If touched scene YAML: verify player/enemy spawns correctly
- If touched script engine: verify Python scripts still execute

Ask yourself: "Does the basic gameplay still work?"

## Step 3: Detect Patterns

Flag these issues:

### 🚩 PATTERN VIOLATIONS (BLOCK COMMIT):
1. **Ground detection via position checking** - use raycast or physics callbacks instead
2. **Magic numbers without explanation** - e.g. `velocity_y < -50` needs a comment
3. **Duplicate Lua/Python script configs** - maintain ONE canonical version
4. **Kinematic body without contact detection** - will have collision issues

### ⚠️ WARNINGS (NOTE BUT DON'T BLOCK):
1. Python script without test file
2. Physics body without clear type comment
3. Scene YAML with hardcoded values that should be config

## Step 4: Check Lua/Python split

Per CLAUDE.md: "Python scripts only" — if you see:
- Updated `.lua` script in metalslug → ERROR, should be Python
- `.py` script but config still points to `.lua` → ERROR, fix config
- Both `level1.yaml` and `level1_python.yaml` → ERROR, consolidate

## Step 5: Report

Output a structured report:

```
═══ GAME-DEV CHECK ═══

[STATUS: ✅ PASS / ❌ FAIL]

TESTS:
  [✅/❌] Python script tests pass
  [✅/❌] No scripts missing test files
  
PATTERNS:
  [✅/❌] No ground detection hacks
  [✅/❌] No duplicate configs
  [✅/❌] Python-only (no Lua updates)

REGRESSIONS:
  [✅/❌] Golden path intact (move/jump/shoot)

ISSUES FOUND:
  • [CRITICAL] Ground detection using position polling (line X)
  • [WARNING] Script missing test file: enemy_walk.py

ACTIONS TO FIX:
  1. Implement raycast for ground detection in physics/box2d
  2. Add enemy_walk_test.py

═══════════════════
```

**If FAIL**: List exactly what to fix and why.
**If PASS**: "All checks passed. Safe to commit."

## Rules

- If any test fails → FAIL entire check
- If golden path broken → FAIL
- If pattern violation → FAIL (block commit)
- Warnings don't block but should be noted
- Be specific about line numbers and file paths
