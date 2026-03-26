# Stem Panic — Implementation Plan (Levels 1-3)

## Context

Stem Panic is a WASM arcade game that teaches kids Kukicha programming concepts through breakout-style gameplay. Instead of destroying bricks, players route a "tea stem" through "processing stations" that transform it. The game lives in `examples/stem-panic/` as a showcase example.

The name is a triple pun: kukicha tea is made from **stems**, the language uses `petiole` (a botanical **stem** connector) for packages, and an unhandled error causes a **panic** — a bitter brew.

## Theme: The Tea Factory

**Premise:** You are a Tea Master in a kukicha processing facility. Route raw stems through processing stations to produce perfect brew. A bad batch causes a **Bitter Panic**.

### Visual Palette (Tea-House Aesthetic)

| Element | Color | RGB |
|---------|-------|-----|
| Background (dark wood) | Deep walnut | `MakeColor(35, 25, 20, 255)` |
| Accent lines (bamboo) | Warm bamboo | `MakeColor(180, 150, 80, 255)` |
| Healthy stem | Matcha green | `MakeColor(100, 180, 60, 255)` |
| Processed/dried leaf | Warm gold | `MakeColor(220, 180, 50, 255)` |
| Bitter/error stem | Deep red | `MakeColor(200, 40, 30, 255)` |
| Paddle (strainer) | Light bamboo | `MakeColor(210, 180, 120, 255)` |
| Shield glow | Jade green | `MakeColor(0, 180, 100, 120)` |
| Station body | Dark clay | `MakeColor(80, 55, 40, 255)` |
| Station label | Cream | `MakeColor(240, 230, 200, 255)` |
| Text | Warm white | `MakeColor(240, 230, 210, 255)` |
| Score/UI | Amber | `MakeColor(220, 160, 40, 255)` |

### Thematic Mappings

| Game Mechanic | Tea Factory Name | Visual |
|---------------|-----------------|--------|
| Paddle | **Bamboo Strainer** | Rectangle with `\|>` label, light bamboo color |
| Ball | **Tea Stem** | Circle, green when raw, gold when processed |
| Pipes | **Processing Stations** | Dark clay rectangles with cream function labels |
| Error ball | **Bitter Leaf** | Red circle with fading trail |
| `onerr` shield | **Quality Filter** | Jade green overlay on strainer + `onerr` label |
| Score | **Brew Quality** | Amber text, top-right |
| Game over | **Bitter Panic** | Screen flashes red, shows panic stack trace |
| Level complete | **Batch Complete** | Gold text, steam effect (expanding circles) |
| Title screen | **STEM PANIC** | Large text + "A kukicha tea processing game" |

## File Structure

```
examples/stem-panic/
    main.kuki       # All game code (~500 lines)
    serve.kuki      # HTTP server for local WASM testing
```

Single file — this is a teaching example. Kids should read it top to bottom.

## Implementation Steps

### Step 1: Scaffold + Title Screen
Create `main.kuki` with window setup, constants, and a title screen scene.
- `petiole main`, imports (`stdlib/game`, `fmt`)
- Color constants: tea-house palette (dark wood bg, warm whites, bamboo accents)
- `variable scene string = "title"`
- Title screen: "STEM PANIC" in warm white + "A kukicha tea processing game" subtitle
- "Press ENTER to start" blinking via `FrameCount() % 60 < 40`
- Decorative bamboo accent lines along top/bottom edges
- Create `serve.kuki` (copy pattern from tutorial lesson 01)
- **Verify:** `kukicha build --wasm main.kuki`, serve, see title in browser

### Step 2: Strainer + Stem Physics
Add the Bamboo Strainer (paddle) and Tea Stem (ball) with basic physics.
- Strainer: `game.Rect`, light bamboo color, moves with arrow keys, `|>` label via `DrawText`
- Stem: `game.Circle`, matcha green, velocity-based movement, wall bouncing
- Strainer collision with angle-based deflection (from breakout tutorial)
- Stem drop = reset to center above strainer
- Dark wood background with subtle bamboo border lines
- **Verify:** Stem bounces, strainer moves, stem resets on drop

### Step 3: State Machine
Wire up scene transitions via string + switch dispatch.
- Scenes: `"title"` -> `"playing"` -> `"batch_complete"` / `"bitter_panic"` -> `"title"`
- Separate `updateTitle()`, `updatePlaying()`, `drawTitle()`, `drawPlaying()`, etc.
- Bitter Panic screen: red flash + panic stack trace aesthetic:
  ```
  BITTER PANIC: unhandled contamination
  onerr panic "batch ruined"
  Press SPACE to retry
  ```
- Batch Complete screen: "BATCH COMPLETE" in gold + expanding circle "steam" effect
- **Verify:** Navigate title -> play -> bitter panic -> title

### Step 4: Processing Station + Suck-and-Shoot Mechanic (Level 1 Core)
The key differentiator from breakout. One station labeled `steam.Process`.
- `type Station` with position, funcName, active flag, intake/exit zones
- Intake collision: `CircleOverlapsRect` on station's top opening
- Suck animation: stem follows linear path through station over 30 frames
- Transform: stem changes color (green -> gold) and grows (radius 5 -> 8)
- "Brew flash" effect: expanding warm-gold circle at exit point
- Eject stem downward with new velocity
- Station drawn as dark clay rectangle with cream label text
- **Verify:** Stem enters station top, animates through, exits transformed

### Step 5: Level 1 — "First Steep"
- Station reactivates after 120 frames (2 seconds)
- Win after 3 successful station passes
- Batch Complete screen: "FIRST STEEP COMPLETE" + brew quality score
- Stem speed is slow (2.0), strainer is wide — nearly impossible to lose
- Teaching moment: the `|>` pipe operator transforms data, just like `steam.Process` transforms raw stems
- **Verify:** Play through Level 1 start to finish

### Step 6: Quality Filter Mechanic
- Hold Spacebar -> strainer shows jade green overlay + `onerr` label
- Filter state tracked as `variable filtered bool`
- No gameplay effect yet (Level 2 activates it)
- Visual: semi-transparent green rectangle over strainer
- **Verify:** Visual filter toggles with spacebar

### Step 7: Level 2 — "Quality Control"
Three flaky stations (`roll`, `dry`, `sort`).
- 30% chance station ejects a bitter leaf (red, `isBitter = true`)
- Unfiltered bitter leaf hits strainer -> Bitter Panic:
  ```
  BITTER PANIC: unhandled contamination
  onerr panic "batch quality failed"
  Press SPACE to retry
  ```
- Filtered bitter hit -> converts stem back to green (healthy), +100 brew quality
- Teaching moment: `onerr` catches errors before they crash your program
- Win: clear all 3 stations (each hit once successfully)
- Bitter leaf trail: 2-3 fading red circles behind it
- **Verify:** Bitter leaves appear randomly, filter saves, unfiltered = panic screen

### Step 8: Level Select (Tea Menu)
- Scene `"tea_menu"` with 3 entries (arrow keys + ENTER)
- Track `variable highestUnlocked int = 1`
- Unlocked levels in warm white, locked in dark gray
- Level names displayed: "First Steep", "Quality Control", "The Full Blend"
- Tea-menu aesthetic: list looks like a traditional menu
- **Verify:** Navigate between levels, only unlocked ones selectable

### Step 9: Level 3 — "The Full Blend"
Four stations stacked vertically: `harvest` -> `steam` -> `dry` -> `separate`.
- This mirrors the real kukicha tea processing order: harvesting, steaming (kill-green), drying/rolling, and separation of stems from leaves
- `variable nextInChain int = 0` tracks required sequence
- Hitting correct station: suck + transform. Hitting wrong: bounce + reset chain
- Current target station pulses (brightness alternates using `FrameCount()`)
- Completed stations in chain show a checkmark label
- Teaching moment: function chaining — data flows through `|>` in sequence
- Win: complete the full chain 2 times (4 stations per chain)
- **Verify:** Must hit in order, wrong order resets, completing chain wins

### Step 10: Polish
- Stem color by state: Green=Raw, Gold=Processed, Red=Bitter
- Station labels drawn inside clay rectangles with `DrawText`
- Brew Quality score display top-right in amber
- Fading red trail for bitter leaves (2-3 trailing circles)
- Bamboo accent lines framing the play area
- Subtle "steam" particles: small circles that drift up from completed stations

## Core Types

```kukicha
type Stem
    body game.Circle
    dx float64
    dy float64
    state string          # "raw", "processed", "bitter"
    isBitter bool
    isSucked bool
    suckTimer int
    suckStationIndex int

type Station
    rect game.Rect
    funcName string       # "steam.Process", "wilt", "sort", etc.
    active bool
    isFlaky bool          # Level 2: can produce bitter output
    sequence int          # Level 3: required hit order
    cooldown int          # Level 1: reactivation timer
    completed bool        # Level 3: chain progress
```

## Critical Files
- `stdlib/game/game.kuki` — full game API surface
- `docs/tutorials/game/08-breakout.md` — strainer/stem/collision reference patterns
- `docs/tutorials/game/06-score-and-state.md` — state management patterns
- `docs/GAME-PLAN.md` — full 10-level game specification (generic names)

## Verification
- `kukicha check main.kuki` after each step (syntax validation)
- `kukicha build --wasm main.kuki` (compiles to WASM)
- Manual browser testing with `serve.kuki` at each step
- Each step produces a playable state (incremental delivery)
