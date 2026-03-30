# Lesson 6: Score and Game State

Add scoring, a game-over condition, and restart logic.

## What you'll learn

- `int` state tracking (score, lives)
- `and` / `else` for complex conditions
- `game.IsKeyPressed` for one-shot key detection
- `game.Random` for randomized gameplay

## The code

```kukicha
# score.kuki

import "stdlib/game"

variable player game.Rect = game.Rect{X: 300, Y: 400, Width: 30, Height: 30}
variable target game.Rect = game.Rect{X: 200, Y: 50, Width: 20, Height: 20}
variable targetSpeed float64 = 2
variable score int = 0
variable lives int = 3
variable gameOver bool = false

function main()
    _ = game.Window("Catch Game", 640, 480)
        |> game.OnUpdate(update)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function update()
    if gameOver
        if game.IsKeyPressed(game.KeySpace)
            restart()
        return

    movePlayer()

    # Target falls down
    target.Y = target.Y + targetSpeed

    # Target reached the bottom — lose a life
    if target.Y > 480
        lives = lives - 1
        if lives <= 0
            gameOver = true
        resetTarget()

    # Player caught the target
    if game.Overlaps(player, target)
        score = score + 1
        if score % 5 equals 0 and targetSpeed < 8.0
            targetSpeed = targetSpeed + 0.5
        resetTarget()

function movePlayer()
    if game.IsKeyDown(game.KeyLeft) and player.X > 0
        player.X = player.X - 5
    if game.IsKeyDown(game.KeyRight) and player.X < 610
        player.X = player.X + 5

function resetTarget()
    target.X = game.Random(20, 600) as float64
    target.Y = -20.0

function restart()
    score = 0
    lives = 3
    targetSpeed = 2.0
    gameOver = false
    resetTarget()

function draw(screen game.Screen)
    game.Clear(screen, game.MakeColor(20, 20, 50, 255))

    if gameOver
        game.DrawText(screen, "Game Over! Score: {score}", 230, 200, game.Red)
        game.DrawText(screen, "Press SPACE to restart", 220, 230, game.White)
        return

    game.DrawRect(screen, player.X, player.Y, player.Width, player.Height, game.Green)
    game.DrawRect(screen, target.X, target.Y, target.Width, target.Height, game.Yellow)
    game.DrawText(screen, "Score: {score}  Lives: {lives}", 10, 10, game.White)
```

## How it works

1. `score` and `lives` are file-scope variables that persist across frames.
2. The target falls from the top. Catching it adds a point; missing costs a life.
3. `and` combines conditions: move left only if the key is down **and** we're not at the edge.
4. `game.IsKeyPressed` fires once per press — unlike `IsKeyDown` which fires every frame.
5. Every 5 points, `targetSpeed` increases, making the game harder. The `%` operator gives the **remainder** after division — `score % 5 equals 0` is true when score is 5, 10, 15, etc.
6. When `gameOver` is true, the update function skips game logic and waits for SPACE.

## IsKeyDown vs IsKeyPressed

| Function | Fires when |
|----------|-----------|
| `game.IsKeyDown(key)` | Key is held down (fires every frame) |
| `game.IsKeyPressed(key)` | Key was just pressed this frame (fires once) |

Use `IsKeyDown` for movement (continuous), `IsKeyPressed` for actions (one-shot).

## Try it

- Add bonus targets worth extra points (different color, different size)
- Make the target move horizontally too while falling
- Add a high score that persists across restarts
