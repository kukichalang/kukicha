# Lesson 4: Animation

A bouncing ball that moves on its own.

## What you'll learn

- `constant` for values that never change (beginner alias for `const`)
- Arithmetic with `+`, `-`, `*`
- `or` for combining conditions (Kukicha's version of `||`)

## The code

```kukicha
# bounce.kuki

import "stdlib/game"

constant screenWidth = 640
constant screenHeight = 480
constant ballRadius = 15.0

variable ballX float64 = 320
variable ballY float64 = 240
variable velocityX float64 = 3
variable velocityY float64 = 2

function main()
    _ = game.Window("Bouncing Ball", screenWidth, screenHeight)
        |> game.OnUpdate(update)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function update()
    # Move the ball
    ballX = ballX + velocityX
    ballY = ballY + velocityY

    # Bounce off left or right walls
    if ballX - ballRadius < 0.0 or ballX + ballRadius > screenWidth as float64
        velocityX = velocityX * -1.0

    # Bounce off top or bottom walls
    if ballY - ballRadius < 0.0 or ballY + ballRadius > screenHeight as float64
        velocityY = velocityY * -1.0

function draw(screen game.Screen)
    game.Clear(screen, game.MakeColor(20, 20, 40, 255))
    game.DrawCircle(screen, ballX, ballY, ballRadius, game.Yellow)
```

## How it works

1. `constant` defines values that never change — the compiler prevents reassignment.
2. Each frame, `update` adds velocity to position, making the ball move.
3. When the ball hits a wall, we reverse the velocity (`* -1`) to make it bounce.
4. `screenWidth as float64` converts the integer constant to a decimal number so it can be compared with `ballX` (which is `float64`). Kukicha doesn't mix types automatically — you use `as` to convert explicitly.
5. `or` combines two conditions: bounce if the ball hits the left **or** right wall.

## The animation loop

```
Frame 1: ballX = 320, velocityX = 3  →  ballX = 323
Frame 2: ballX = 323, velocityX = 3  →  ballX = 326
...
Frame N: ballX = 638  →  hit right wall  →  velocityX = -3
Frame N+1: ballX = 635  →  moving left now
```

## Try it

- Change the velocity values to make the ball faster or slower
- Add a second ball with its own position and velocity variables
- Change the ball color based on direction:
  ```kukicha
  if velocityX > 0
      game.DrawCircle(screen, ballX, ballY, ballRadius, game.Red)
  else
      game.DrawCircle(screen, ballX, ballY, ballRadius, game.Blue)
  ```
