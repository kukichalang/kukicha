# Lesson 8: Breakout!

Build a complete Breakout game using everything you've learned.

## What you'll learn

- `list of` for collections of game objects
- `for` loops to update and draw many objects
- `len` and `append` for managing lists
- `game.CircleOverlapsRect` for ball-to-brick collision

## The code

```kukicha
# breakout.kuki

import "stdlib/game"

constant screenW = 640
constant screenH = 480
constant paddleW = 80.0
constant paddleH = 12.0
constant brickW = 58.0
constant brickH = 18.0
constant ballRadius = 6.0

variable paddle game.Rect
variable ball game.Circle
variable ballDX float64 = 3.0
variable ballDY float64 = -3.0
variable bricks list of game.Rect
variable alive list of bool
variable score int = 0
variable gameOver bool = false
variable won bool = false

function main()
    _ = game.Window("Breakout!", screenW, screenH)
        |> game.OnSetup(setup)
        |> game.OnUpdate(update)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function setup()
    paddle = game.Rect{X: 280.0, Y: 450.0, Width: paddleW, Height: paddleH}
    ball = game.Circle{X: 320.0, Y: 430.0, Radius: ballRadius}

    # Create a grid of bricks
    bricks = make(list of game.Rect, 0)
    alive = make(list of bool, 0)
    for row from 0 to 5
        for col from 0 to 10
            x := 12.0 + col as float64 * 62.0
            y := 40.0 + row as float64 * 24.0
            bricks = append(bricks, game.Rect{X: x, Y: y, Width: brickW, Height: brickH})
            alive = append(alive, true)

function update()
    if gameOver or won
        if game.IsKeyPressed(game.KeySpace)
            score = 0
            gameOver = false
            won = false
            setup()
        return

    # Move paddle
    if game.IsKeyDown(game.KeyLeft) and paddle.X > 0.0
        paddle.X = paddle.X - 6.0
    if game.IsKeyDown(game.KeyRight) and paddle.X + paddleW < screenW as float64
        paddle.X = paddle.X + 6.0

    # Move ball
    ball.X = ball.X + ballDX
    ball.Y = ball.Y + ballDY

    # Bounce off walls
    if ball.X - ballRadius < 0.0 or ball.X + ballRadius > screenW as float64
        ballDX = ballDX * -1.0
    if ball.Y - ballRadius < 0.0
        ballDY = ballDY * -1.0

    # Ball fell below screen
    if ball.Y > screenH as float64
        gameOver = true
        return

    # Bounce off paddle
    if game.CircleOverlapsRect(ball, paddle) and ballDY > 0.0
        ballDY = ballDY * -1.0
        # Angle the bounce based on where the ball hit the paddle
        hitPos := (ball.X - paddle.X) / paddleW
        ballDX = (hitPos - 0.5) * 6.0

    # Check brick collisions
    remaining := 0
    for i from 0 to len(bricks)
        if not alive[i]
            continue
        remaining = remaining + 1
        if game.CircleOverlapsRect(ball, bricks[i])
            alive[i] = false
            ballDY = ballDY * -1.0
            score = score + 10

    if remaining equals 0
        won = true

function draw(screen game.Screen)
    game.Clear(screen, game.MakeColor(10, 10, 30, 255))

    if gameOver
        game.DrawText(screen, "Game Over! Score: {score}", 230, 220, game.Red)
        game.DrawText(screen, "Press SPACE to restart", 220, 250, game.White)
        return

    if won
        game.DrawText(screen, "You Win! Score: {score}", 240, 220, game.Green)
        game.DrawText(screen, "Press SPACE to play again", 210, 250, game.White)
        return

    # Draw paddle
    game.DrawRect(screen, paddle.X, paddle.Y, paddle.Width, paddle.Height, game.White)

    # Draw ball
    game.DrawCircle(screen, ball.X, ball.Y, ball.Radius, game.Yellow)

    # Draw bricks with row-based colors
    colors := list of game.Color{
        game.Red,
        game.Orange,
        game.Yellow,
        game.Green,
        game.Blue,
    }
    for i from 0 to len(bricks)
        if not alive[i]
            continue
        row := i / 10
        c := colors[row % len(colors)]
        game.DrawRect(screen, bricks[i].X, bricks[i].Y, bricks[i].Width, bricks[i].Height, c)

    # Draw score
    game.DrawText(screen, "Score: {score}", 10, 10, game.White)
```

## How it works

### Lists

```kukicha
bricks = make(list of game.Rect, 0)     # Create an empty list
bricks = append(bricks, newBrick)        # Add an element
len(bricks)                              # Count of elements
bricks[i]                                # Access by index
```

### The game loop

1. **Setup**: Create a 5x10 grid of bricks, position the paddle and ball
2. **Update**: Move ball, check wall/paddle/brick collisions, detect win/lose
3. **Draw**: Render everything — bricks get row-based colors. `i / 10` is **integer division** (dividing two whole numbers drops the remainder, so bricks 0–9 give row 0, bricks 10–19 give row 1, etc.). `row % len(colors)` uses the modulo operator to cycle through the color list — it wraps back to 0 when the row exceeds the number of colors

### CircleOverlapsRect

`game.CircleOverlapsRect(circle, rect)` finds the closest point on the rectangle to the circle's center, then checks if that distance is less than the radius. This gives accurate collision between the round ball and rectangular bricks/paddle.

## Concepts used from all lessons

| Lesson | Concept | Used here |
|--------|---------|-----------|
| 1 | `function`, pipes, `OnDraw` | Game structure |
| 2 | Drawing shapes, colors | Rendering |
| 3 | `variable`, `if`, `IsKeyDown` | Paddle movement |
| 4 | `constant`, animation, `or` | Ball physics |
| 5 | `game.Rect`, `Overlaps` | Collision detection |
| 6 | Score, game state, `and` | Game logic |
| 7 | `OnSetup`, initialization | Level setup |
| 8 | `list of`, `for`, `len` | Brick management |

## Try it

- Add multiple lives (3 balls before game over)
- Make bricks worth different points per row
- Add power-ups that fall when bricks break (wider paddle, multi-ball)
- Add a second level with a different brick layout

## What's next?

Want to see a larger game built with Kukicha? Check out [Stem Panic](https://github.com/kukichalang/game/examples/stem-panic/) — a tapper-style tea processing game that uses multiple lanes, game states, levels, and the `|>` pipe as a gameplay mechanic. It builds on all the concepts from these tutorials.
