# Lesson 7: Setup and Polish

Use `OnSetup` for initialization and add visual polish to your game.

## What you'll learn

- `game.OnSetup` for one-time initialization
- `onerr panic` for handling startup errors
- Organizing game state into setup vs runtime

## The code

```kukicha
# polished.kuki

import "stdlib/game"

variable player game.Rect
variable targets list of game.Rect
variable score int = 0
variable speed float64 = 2.0

function main()
    _ = game.Window("Polished Game", 640, 480)
        |> game.OnSetup(setup)
        |> game.OnUpdate(update)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function setup()
    # Initialize player at the bottom center
    player = game.Rect{X: 300.0, Y: 420.0, Width: 40.0, Height: 20.0}

    # Create initial targets
    targets = make(list of game.Rect, 0)
    for i from 0 to 5
        targets = append(targets, makeTarget())

function makeTarget() game.Rect
    return game.Rect{
        X: game.Random(20, 600) as float64,
        Y: game.Random(-200, -20) as float64,
        Width: 15.0,
        Height: 15.0,
    }

function update()
    # Move player
    if game.IsKeyDown(game.KeyLeft) and player.X > 0.0
        player.X = player.X - 6.0
    if game.IsKeyDown(game.KeyRight) and player.X < 600.0
        player.X = player.X + 6.0

    # Move targets down and check collisions
    for i from 0 to len(targets)
        targets[i].Y = targets[i].Y + speed
        if game.Overlaps(player, targets[i])
            score = score + 1
            targets[i] = makeTarget()
        if targets[i].Y > 500.0
            targets[i] = makeTarget()

function draw(screen game.Screen)
    game.Clear(screen, game.MakeColor(15, 15, 35, 255))

    # Draw player
    game.DrawRect(screen, player.X, player.Y, player.Width, player.Height, game.Green)

    # Draw targets
    for i from 0 to len(targets)
        game.DrawRect(screen, targets[i].X, targets[i].Y, targets[i].Width, targets[i].Height, game.Yellow)

    # Draw score
    game.DrawText(screen, "Score: {score}", 10, 10, game.White)
```

## How it works

1. `game.OnSetup(setup)` runs `setup()` once before the game loop starts.
2. Inside `setup`, we initialize the player position and create an initial list of targets. `make(list of game.Rect, 0)` creates an empty list — the `0` is the starting capacity (how many items to pre-allocate space for).
3. The `makeTarget()` helper creates a target at a random X position above the screen.
4. When a target is caught or falls off-screen, it's recycled with `makeTarget()`.

## Why use OnSetup?

Without `OnSetup`, you'd need to initialize everything at the file scope with `variable`. That works for simple games, but `OnSetup` is better because:

- Complex initialization (lists, loops) is cleaner in a function
- Error handling is possible (for loading assets later)
- Clear separation between "configure" and "initialize"

## Try it

- Increase `speed` every 10 points
- Add different colored targets worth different points
- Add a particle effect when collecting (draw small circles that fade out)
