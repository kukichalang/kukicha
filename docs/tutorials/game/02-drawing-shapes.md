# Lesson 2: Drawing Shapes

Draw rectangles, circles, and lines with different colors.

## What you'll learn

- Local variables with `:=`
- Built-in color constants (`game.Red`, `game.Blue`, etc.)
- `game.DrawRect`, `game.DrawCircle`, `game.DrawLine`

## The code

```kukicha
# shapes.kuki

import "stdlib/game"

function main()
    _ = game.Window("Drawing Shapes", 640, 480)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function draw(screen game.Screen)
    game.Clear(screen, game.Black)

    # Draw a red rectangle
    game.DrawRect(screen, 50, 50, 200, 100, game.Red)

    # Draw a blue circle
    game.DrawCircle(screen, 400, 150, 60, game.Blue)

    # Draw a green line
    game.DrawLine(screen, 100, 300, 500, 350, game.Green)

    # Draw a yellow rectangle
    game.DrawRect(screen, 300, 300, 150, 120, game.Yellow)

    # Draw a custom-colored circle
    sky := game.MakeColor(100, 180, 255, 255)
    game.DrawCircle(screen, 320, 100, 40, sky)
```

## How it works

Each draw function takes the screen, position/size, and a color:

- `DrawRect(screen, x, y, width, height, color)` — top-left corner at (x, y)
- `DrawCircle(screen, x, y, radius, color)` — center at (x, y)
- `DrawLine(screen, x1, y1, x2, y2, color)` — from (x1,y1) to (x2,y2)

The coordinate system starts at the top-left corner (0, 0). X increases to the right, Y increases downward.

## Try it

- Add more shapes to create a simple scene (house, tree, sun)
- Create your own colors with `game.MakeColor(r, g, b, a)`
- Use `game.Orange`, `game.Purple`, or `game.Gray`

## Available colors

| Constant | Color |
|----------|-------|
| `game.Red` | Red |
| `game.Green` | Green |
| `game.Blue` | Blue |
| `game.White` | White |
| `game.Black` | Black |
| `game.Yellow` | Yellow |
| `game.Orange` | Orange |
| `game.Purple` | Purple |
| `game.Gray` | Gray |
