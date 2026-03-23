# Lesson 5: Collision Detection

A player that collects a target — your first interactive game mechanic.

## What you'll learn

- `game.Rect` type for rectangles with collision
- `game.Overlaps` for detecting when two rectangles touch
- Helper functions to organize your code
- `bool` return type

## The code

```kukicha
# collect.kuki

import "stdlib/game"

variable player game.Rect = game.Rect{X: 300, Y: 200, Width: 30, Height: 30}
variable target game.Rect = game.Rect{X: 100, Y: 100, Width: 20, Height: 20}

function main()
    _ = game.Window("Collect the Target", 640, 480)
        |> game.OnUpdate(update)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function update()
    movePlayer()
    if game.Overlaps(player, target)
        respawnTarget()

function movePlayer()
    if game.IsKeyDown(game.KeyLeft)
        player.X = player.X - 4
    if game.IsKeyDown(game.KeyRight)
        player.X = player.X + 4
    if game.IsKeyDown(game.KeyUp)
        player.Y = player.Y - 4
    if game.IsKeyDown(game.KeyDown)
        player.Y = player.Y + 4

function respawnTarget()
    target.X = game.Random(20, 600) as float64
    target.Y = game.Random(20, 440) as float64

function draw(screen game.Screen)
    game.Clear(screen, game.Black)
    game.DrawRect(screen, player.X, player.Y, player.Width, player.Height, game.Green)
    game.DrawRect(screen, target.X, target.Y, target.Width, target.Height, game.Red)
    game.DrawText(screen, "Collect the red squares!", 10, 10, game.White)
```

## How it works

1. Both `player` and `target` are `game.Rect` values — they have X, Y, Width, Height.
2. `game.Overlaps(a, b)` returns `true` when two rectangles overlap.
3. When the player touches the target, `respawnTarget()` moves it to a random position.
4. `game.Random(20, 600)` returns a random integer between 20 and 599.

## AABB collision

`Overlaps` uses **AABB** (Axis-Aligned Bounding Box) collision — it checks if the edges of two rectangles overlap on both axes:

```
a.X < b.X + b.Width    (a's left is before b's right)
a.X + a.Width > b.X    (a's right is after b's left)
a.Y < b.Y + b.Height   (a's top is before b's bottom)
a.Y + a.Height > b.Y   (a's bottom is after b's top)
```

All four must be true for a collision.

## Try it

- Make the target smaller for a harder challenge
- Add a timer or frame counter to track how fast you collect targets
- Change the player to a circle using `game.DrawCircle`
