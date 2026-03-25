# Pipeline Panic — Implementation Plan (Levels 1-3)

## Context

Pipeline Panic is a WASM arcade game that teaches kids Kukicha programming concepts through breakout-style gameplay. Instead of destroying bricks, players route a "data ball" through "stdlib pipes" that transform it. The game lives in `examples/pipeline-panic/` as a showcase example. It builds on the existing breakout tutorial's paddle/ball mechanics but adds pipe transformations, error handling (onerr shield), and sequential function chaining.

The user selected: levels 1-3 only, in the main repo, with game library extensions anticipated but not blocking level 1-3 delivery.

## Key Decision: No Game Library Extensions Needed for Levels 1-3

The existing `kukichalang/game` API (DrawRect, DrawCircle, DrawText, collision, input) is sufficient. The terminal aesthetic (dark background, monospace debug font, geometric shapes) actually suits the "Data Plumber" theme. Sound/sprites/particles can be added later when extending to levels 4-10.

## File Structure

```
examples/pipeline-panic/
    main.kuki       # All game code (~500 lines)
    serve.kuki      # HTTP server for local WASM testing
```

Single file — this is a teaching example. Kids should read it top to bottom.

## Implementation Steps

### Step 1: Scaffold + Title Screen
Create `main.kuki` with window setup, constants, and a title screen scene.
- `petiole main`, imports (`stdlib/game`, `fmt`)
- Constants: screen size (640x480), colors (terminal dark bg `MakeColor(20,22,25,255)`)
- `variable scene string = "title"`
- Title screen: "PIPELINE PANIC" + "Press ENTER to start" via `DrawText`
- Create `serve.kuki` (copy pattern from tutorial lesson 01)
- **Verify:** `kukicha build --wasm main.kuki`, serve, see title in browser

### Step 2: Paddle + Ball Physics
Add the Operator (paddle) and Payload (ball) with basic physics.
- Paddle: `game.Rect`, moves with arrow keys, draws with `DrawRect` + `|>` label
- Ball: `game.Circle`, velocity-based movement, wall bouncing
- Paddle collision with angle-based deflection (from breakout tutorial)
- Ball drop = reset to center above paddle
- **Verify:** Ball bounces, paddle moves, ball resets on drop

### Step 3: State Machine
Wire up scene transitions via string + switch dispatch.
- Scenes: `"title"` → `"playing"` → `"level_complete"` / `"game_over"` → `"title"`
- Separate `updateTitle()`, `updatePlaying()`, `drawTitle()`, `drawPlaying()`, etc.
- **Verify:** Navigate title → play → game over → title

### Step 4: Pipe Object + Suck-and-Shoot Mechanic (Level 1 Core)
The key differentiator from breakout. One pipe labeled `string.ToUpper`.
- `type Pipe` with position, funcName, active flag, intake/exit zones
- Intake collision: `CircleOverlapsRect` on pipe's top opening
- Suck animation: ball follows linear path through pipe over 30 frames
- Transform: ball changes color (white→blue) and grows (radius 5→8)
- "Compile flash" effect: expanding white circle at exit point
- Eject ball downward with new velocity
- **Verify:** Ball enters pipe top, animates through, exits transformed

### Step 5: Level 1 Completion
- Pipe reactivates after 120 frames (2 seconds)
- Win after 3 successful pipe passes
- Level complete screen: "LEVEL 1 COMPLETE" + "Press ENTER"
- Ball speed is slow (2.0), paddle is wide — nearly impossible to lose
- **Verify:** Play through Level 1 start to finish

### Step 6: Shield Mechanic
- Hold Spacebar → paddle shows green overlay (`MakeColor(0,200,0,100)`) + `onerr` label
- Shield state tracked as `variable shielded bool`
- No gameplay effect yet (Level 2 activates it)
- **Verify:** Visual shield toggles with spacebar

### Step 7: Level 2 — "The Bug Hunter"
Three flaky pipes (`parse`, `validate`, `format`).
- 30% chance pipe ejects an error ball (red, `isError = true`)
- Unshielded error paddle hit → game over with panic screen:
  ```
  PANIC: unhandled error
  onerr panic "data validation failed"
  Press SPACE to retry
  ```
- Shielded error hit → converts ball back to white (healthy), +100 score
- Win: clear all 3 pipes (each hit once successfully)
- **Verify:** Errors appear randomly, shield saves, unshielded = panic screen

### Step 8: Level Select
- Scene `"level_select"` with 3 entries (arrow keys + ENTER)
- Track `variable highestUnlocked int = 1`
- Locked levels shown in gray
- **Verify:** Navigate between levels, only unlocked ones selectable

### Step 9: Level 3 — "Function Chaining"
Three pipes stacked vertically: `fetch` → `JSON` → `Clean`.
- `variable nextInChain int = 0` tracks required sequence
- Hitting correct pipe: suck + transform. Hitting wrong pipe: bounce + reset chain
- Current target pipe pulses (brightness alternates using `FrameCount()`)
- Completed pipes in chain show checkmark text
- Win: complete the full chain 2 times
- **Verify:** Must hit in order, wrong order resets, completing chain wins

### Step 10: Polish
- Ball color by type: White=String, Yellow=Int, Blue=Transformed, Red=Error
- Pipe labels drawn inside pipe rectangles with `DrawText`
- Score display top-right
- Fading red trail for error balls (2-3 trailing circles)
- Terminal aesthetic colors throughout

## Core Types

```kukicha
type Payload
    body game.Circle
    dx float64
    dy float64
    typeName string       # "raw", "string", "int", "error"
    isError bool
    isSucked bool
    suckTimer int
    suckPipeIndex int

type Pipe
    rect game.Rect
    funcName string
    active bool
    isFlaky bool
    sequence int          # Level 3: required hit order
    cooldown int          # Level 1: reactivation timer
```

## Critical Files
- `stdlib/game/game.kuki` — full game API surface
- `docs/tutorials/game/08-breakout.md` — paddle/ball/collision reference patterns
- `docs/tutorials/game/06-score-and-state.md` — state management patterns
- `docs/GAME-PLAN.md` — full game specification

## Verification
- `kukicha check main.kuki` after each step (syntax validation)
- `kukicha build --wasm main.kuki` (compiles to WASM)
- Manual browser testing with `serve.kuki` at each step
- Each step produces a playable state (incremental delivery)
