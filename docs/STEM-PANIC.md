# Stem Panic — Implementation Plan (Tapper-Style Pipeline Game)

## Context

Stem Panic is a WASM arcade game that teaches kids Kukicha programming concepts through Tapper-style gameplay. You are a Tea Master sliding raw stems down conveyor lanes — each lane is a `|>` pipeline with processing stations inline. Match the right pipeline to incoming tea orders before time runs out.

The game lives in `examples/stem-panic/` as a showcase example.

The name is a triple pun: kukicha tea is made from **stems**, the language uses `petiole` (a botanical **stem** connector) for packages, and an unhandled error causes a **panic** — a bitter brew.

## Theme: The Tea Factory

**Premise:** You are a Tea Master at the left end of a kukicha processing facility. Tea orders arrive on the right. Slide raw stems down the correct conveyor lane so they get processed into the right tea. A wrong delivery or unhandled bitter leaf causes a **Bitter Panic**.

### Why Tapper?

Tapper's core loop maps directly to pipes:

| Tapper | Stem Panic | Kukicha Concept |
|--------|-----------|-----------------|
| Bar counter | Conveyor lane | A `\|>` pipeline |
| Slide a beer | Send a raw stem | Pipe input into a function chain |
| Customer wants a drink | Tea order arrives | Expected output type |
| Wrong drink | Wrong pipeline chosen | Type mismatch / wrong function |
| Bartender moves between counters | Tea Master moves between lanes | Choosing which pipeline to use |

### Visual Palette (Tea-House Aesthetic)

| Element | Color | RGB |
|---------|-------|-----|
| Background (dark wood) | Deep walnut | `MakeColor(35, 25, 20, 255)` |
| Conveyor belt | Warm bamboo | `MakeColor(180, 150, 80, 255)` |
| Pipe symbol `\|>` | Bamboo (bright) | `MakeColor(200, 170, 90, 255)` |
| Raw stem | Matcha green | `MakeColor(100, 180, 60, 255)` |
| Processed stem | Warm gold | `MakeColor(220, 180, 50, 255)` |
| Bitter stem | Deep red | `MakeColor(200, 40, 30, 255)` |
| Tea Master | Light bamboo | `MakeColor(210, 180, 120, 255)` |
| onerr catcher | Jade green | `MakeColor(0, 180, 100, 120)` |
| Station body | Dark clay | `MakeColor(80, 55, 40, 255)` |
| Station label | Cream | `MakeColor(240, 230, 200, 255)` |
| Text | Warm white | `MakeColor(240, 230, 210, 255)` |
| Score/UI | Amber | `MakeColor(220, 160, 40, 255)` |
| Order timer bar | Amber -> Red as time runs out |
| Lane highlight (selected) | Slightly brighter wood | `MakeColor(50, 38, 30, 255)` |

### Thematic Mappings

| Game Mechanic | Tea Factory Name | Visual |
|---------------|-----------------|--------|
| Player | **Tea Master** | Rectangle on left edge with `\|>` label, light bamboo |
| Conveyor lane | **Pipeline** | Horizontal bamboo line spanning the screen |
| Station | **Processing Station** | Dark clay rectangle with cream function label |
| Sliding stem | **Tea Stem** | Circle, green when raw, gold when processed |
| Order | **Tea Order** | Rectangle on right edge with order text |
| Order timer | **Patience** | Shrinking amber bar above order |
| Flaky station | **Unreliable Station** | Station with `~err` label |
| onerr catcher | **Quality Filter** | Jade green rectangle after flaky station |
| Bitter stem | **Bitter Leaf** | Red circle with fading trail |
| Score | **Brew Quality** | Amber text, top-right |
| Game over | **Bitter Panic** | Screen flashes red, shows panic stack trace |
| Level complete | **Batch Complete** | Gold text, steam effect |
| Title screen | **STEM PANIC** | Large text + "A kukicha tea processing game" |

## Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  Brew Quality: 450          First Steep  1/3          Lives: 3  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [|>]  ════════[steam]══════════════════════════════  [steamed] │
│  Tea                                                    ████   │
│  Master  ══════[dry]════════════════════════════════  [dried]   │
│  [|>]                                                   ██     │
│         ═══════[steam]═══|>═══[dry]═════════════════  [blend]  │
│                                                         ████   │
│                                                                 │
│  |> pipe sends data through functions                           │
├─────────────────────────────────────────────────────────────────┤
│  ═══bamboo═══                                    ═══bamboo═══  │
└─────────────────────────────────────────────────────────────────┘
```

- Tea Master sits on the left, moves UP/DOWN between lanes
- ENTER or SPACE sends a raw stem sliding rightward
- Stems pause briefly at each station (suck animation), transform, continue
- Orders sit on the right edge with a timer bar
- Correct delivery = order fulfilled, score up
- Wrong tea or bitter tea reaching order = Bitter Panic

## File Structure

```
examples/stem-panic/
    main.kuki       # All game code
    serve.kuki      # HTTP server for local WASM testing
```

Single file — this is a teaching example. Kids should read it top to bottom.

## Implementation Steps

### Step 1: Scaffold + Title Screen
Create `main.kuki` with window setup, constants, and a title screen scene.
- `petiole main`, imports (`stdlib/game`, `fmt`)
- Color constants: tea-house palette
- `variable scene string = "title"`
- Title screen: "STEM PANIC" in warm white + "A kukicha tea processing game" subtitle
- "Press ENTER to start" blinking via `FrameCount() % 60 < 40`
- Decorative bamboo accent lines along top/bottom edges
- Small animated stem sliding through a mini station on the title screen (visual teaser)
- Create `serve.kuki` (copy pattern from tutorial lesson 01)
- **Verify:** `kukicha build --wasm main.kuki`, serve, see title in browser

### Step 2: Lanes + Tea Master
Draw the conveyor lanes and a moveable Tea Master.
- Constants: `laneCount = 3`, `laneStartX = 80`, `laneEndX = 560`, lane Y positions evenly spaced
- Draw lanes as horizontal bamboo-colored rectangles (thin, ~4px tall)
- `|>` symbols drawn at intervals along each lane
- Tea Master: rectangle at left edge, moves UP/DOWN between lane positions
- Current lane highlighted (slightly brighter background strip)
- Tea Master snaps to lane positions (not free movement)
- **Verify:** Three lanes visible, Tea Master moves between them smoothly

### Step 3: Sending Stems + Sliding Physics
Press ENTER/SPACE to send a stem down the current lane.
- `type Stem` with x, y, lane, dx (slide speed), state (raw/processed/bitter), suckTimer
- Stem starts at Tea Master's position, slides rightward at constant speed
- `variable stems list of Stem` — supports multiple stems in flight
- Green circle for raw stem
- Stem removed when it reaches the right edge
- Limit: one stem per lane at a time (or configurable per level)
- **Verify:** Send stems down lanes, they slide right and disappear

### Step 4: Processing Stations
Stations sit on lanes and transform stems as they pass through.
- `type Station` with x, lane, funcName, active, isFlaky, cooldown
- Station drawn as dark clay rectangle straddling the lane, cream label text
- Intake collision: when stem reaches station's x position
- Suck animation: stem pauses, follows a path through the station over 20 frames
- Transform: stem state changes (raw -> processed), color changes (green -> gold)
- "Brew flash" effect: expanding gold circle at station exit
- Stem continues sliding rightward after processing
- **Verify:** Stem enters station, pauses, transforms, exits gold, keeps sliding

### Step 5: Tea Orders + Scoring (Level 1 Core)
Orders arrive on the right edge and must be fulfilled.
- `type Order` with lane, wantState (e.g., "steamed"), timer, active
- Orders drawn as rectangles on the right edge with text label ("wants: steamed")
- Timer bar above each order — shrinks over time, changes amber -> red
- When a processed stem reaches an order on the matching lane:
  - Correct state: order fulfilled, +50 Brew Quality, order removed
  - Wrong state: Bitter Panic
- When timer expires: customer leaves, lose a life
- Level 1 config: 2 lanes, 1 station each (`steam`, `dry`), slow orders, generous timers
- Win: fulfill 5 orders
- **Verify:** Orders appear, stems fulfill them, score increases

### Step 6: State Machine
Wire up scene transitions.
- Scenes: `"title"` -> `"tea_menu"` -> `"playing"` -> `"batch_complete"` / `"bitter_panic"` -> `"tea_menu"`
- Separate update/draw functions per scene
- Bitter Panic screen with red flash + panic stack trace aesthetic:
  ```
  BITTER PANIC: unhandled contamination
  onerr panic "wrong tea delivered"
  Press SPACE to retry
  ```
- Batch Complete screen: "BATCH COMPLETE" in gold + expanding circle "steam" effect
- **Verify:** Navigate title -> menu -> play -> complete/panic -> menu

### Step 7: Level 1 — "First Steep"
Polish Level 1 as a complete, playable experience.
- 2 lanes: lane 1 has `steam`, lane 2 has `dry`
- Orders alternate between "steamed" and "dried" — slow pace
- Generous timers (8 seconds per order)
- Stem speed is slow (2.0 px/frame)
- Teaching hint at bottom: `|> pipe sends data through functions`
- Win after 5 fulfilled orders
- **Verify:** Play through Level 1 start to finish, feels good

### Step 8: Quality Filter Mechanic (onerr Catchers)
Add flaky stations and onerr catchers for Level 2.
- Flaky stations: 30% chance to output a bitter stem (red) instead of processed
- `type Catcher` with x, lane, active, cooldown — jade green rectangle placed after flaky stations
- Catcher labeled `onerr`
- When bitter stem reaches catcher:
  - If catcher active: converts bitter -> raw (green), +100 Brew Quality
  - If catcher inactive: bitter stem passes through
- When bitter stem reaches an order: Bitter Panic
- Player presses SPACE to activate the catcher on their current lane
- Catcher stays active for 60 frames, then has 60 frame cooldown
- Visual: jade green overlay + `onerr` text when active, dim when on cooldown
- **Verify:** Flaky stations produce bitter stems, catchers catch them, timing matters

### Step 9: Level 2 — "Quality Control"
Three lanes with flaky stations and onerr catchers.
- Lane 1: `roll` (flaky) + onerr catcher
- Lane 2: `sort` (flaky) + onerr catcher
- Lane 3: `dry` (flaky) + onerr catcher
- Orders come faster (5 second timers)
- Stems move at medium speed (3.0 px/frame)
- Player must juggle: move to lane, send stem, watch for bitter output, activate catcher in time
- Teaching hint: `onerr catches errors before they crash`
- Win: fulfill 8 orders
- Bitter Panic message: `onerr panic "bitter leaf not caught"` + hint: "activate onerr catcher with SPACE"
- **Verify:** Bitter leaves appear randomly, catchers save the day, real tension

### Step 10: Level 3 — "The Full Blend"
Multi-station chains that must be hit in sequence.
- 3 lanes with different station chains:
  - Lane 1: `harvest |> steam` (produces "steamed")
  - Lane 2: `harvest |> dry` (produces "dried")
  - Lane 3: `harvest |> steam |> dry |> separate` (produces "blended")
- `|>` symbols drawn between chained stations on the same lane
- Orders now request "steamed", "dried", or "blended" tea
- Stems visibly transform at each station in the chain (green -> yellow-green -> gold -> bright gold)
- Some stations in the chain are flaky (onerr catchers still needed)
- Faster pace (4 second timers), stems at 3.5 px/frame
- Teaching hint: `chain functions in sequence with |>`
- Win: fulfill 10 orders
- **Verify:** Multi-station chains process correctly, order matching works

### Step 11: Tea Menu (Level Select)
- Scene `"tea_menu"` with 3 entries (UP/DOWN + ENTER)
- Track `variable highestUnlocked int = 1`
- Unlocked levels in warm white, locked in dark gray
- Level names: "First Steep", "Quality Control", "The Full Blend"
- Tea-menu aesthetic: list looks like a traditional menu
- `|>` cursor indicator next to selected item
- **Verify:** Navigate between levels, only unlocked ones selectable

### Step 12: Polish
- Stem color progression: Green (raw) -> Gold (processed) -> Red (bitter)
- Multi-step chains show intermediate colors (green -> yellow-green -> gold)
- Conveyor animation: small tick marks that scroll rightward along lanes
- Station labels inside clay rectangles
- Brew Quality score top-right in amber
- Fading red trail for bitter stems (2-3 trailing circles)
- Bamboo accent lines framing the play area
- Small steam particles drifting up from fulfilled orders
- Speed ramp: orders arrive slightly faster as score increases within a level
- Sound-like visual feedback: screen briefly flashes on successful delivery

## Core Types

```kukicha
type Stem
    body game.Circle
    dx float64
    lane int
    state string          # "raw", "processed", "bitter"
    suckTimer int
    suckStationIdx int

type Station
    x float64
    lane int
    funcName string       # "steam", "dry", "roll", etc.
    active bool
    isFlaky bool
    cooldown int

type Catcher
    x float64
    lane int
    active bool
    timer int             # frames remaining active
    cooldown int          # frames before reactivation

type Order
    lane int
    wantState string      # "steamed", "dried", "blended"
    timer int             # frames remaining before customer leaves
    maxTimer int          # initial timer value (for drawing bar)
    fulfilled bool
```

## Key Constants

```kukicha
constant screenW = 640
constant screenH = 480
constant laneCount = 3
constant laneStartX = 80.0       # Tea Master position
constant laneEndX = 560.0        # Order position
constant laneSpacing = 100.0     # Vertical space between lanes
constant laneBaseY = 140.0       # Y of first lane
constant stemRadius = 6.0
constant stationW = 60.0
constant stationH = 40.0
constant suckDuration = 20       # Frames stem pauses in station
```

## Critical Files
- `stdlib/game/game.kuki` — full game API surface
- `docs/tutorials/game/08-breakout.md` — collision reference patterns
- `docs/tutorials/game/06-score-and-state.md` — state management patterns

## Verification
- `kukicha check main.kuki` after each step (syntax validation)
- `kukicha build --wasm main.kuki` (compiles to WASM)
- Manual browser testing with `serve.kuki` at each step
- Each step produces a playable state (incremental delivery)
