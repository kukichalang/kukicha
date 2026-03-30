# Lesson 3: Keyboard Input

Move a square around the screen with arrow keys.

## What you'll learn

- `variable` for file-scope state (beginner alias for `var`)
- `if` statements for conditional logic
- `game.OnUpdate` for game logic
- `game.IsKeyDown` for reading keyboard input

## The code

```kukicha
# keyboard.kuki

import "stdlib/game"

variable playerX float64 = 300
variable playerY float64 = 200

function main()
    _ = game.Window("Keyboard Input", 640, 480)
        |> game.OnUpdate(update)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function update()
    if game.IsKeyDown(game.KeyLeft)
        playerX = playerX - 4
    if game.IsKeyDown(game.KeyRight)
        playerX = playerX + 4
    if game.IsKeyDown(game.KeyUp)
        playerY = playerY - 4
    if game.IsKeyDown(game.KeyDown)
        playerY = playerY + 4

function draw(screen game.Screen)
    game.Clear(screen, game.Black)
    game.DrawRect(screen, playerX, playerY, 40, 40, game.Green)
    game.DrawText(screen, "Arrow keys to move", 10, 10, game.White)
```

## How it works

1. `variable playerX float64 = 300` creates a file-scope variable — it keeps its value between frames. `float64` means a decimal number ("float" = floating-point, "64" = 64-bit precision). We use it instead of `int` (whole numbers) because game positions need sub-pixel accuracy for smooth movement.
2. `game.OnUpdate(update)` registers our `update` function to run every frame (60 times per second).
3. Inside `update`, we check each arrow key with `game.IsKeyDown` and adjust the position.
4. `game.OnDraw(draw)` renders the square at the current position every frame.

The game loop runs: **update** (logic) then **draw** (render), 60 times per second.

## Try it

- Change the speed from `4` to something faster or slower
- Make the player a circle instead of a rectangle
- Add boundaries so the player can't leave the screen:
  ```kukicha
  if playerX < 0
      playerX = 0
  if playerX > 600
      playerX = 600
  ```

## Key constants

| Constant | Key |
|----------|-----|
| `game.KeyLeft` | Left arrow |
| `game.KeyRight` | Right arrow |
| `game.KeyUp` | Up arrow |
| `game.KeyDown` | Down arrow |
| `game.KeySpace` | Spacebar |
| `game.KeyEnter` | Enter |
| `game.KeyEscape` | Escape |
