# Lesson 1: Hello World

Your first Kukicha game — a window with colored text.

## What you'll learn

- `import` to load the game library
- `function` to define your own functions
- Pipes (`|>`) to chain function calls
- `game.Window`, `game.OnDraw`, `game.Clear`, `game.DrawText`, `game.Run`

## The code

```kukicha
# hello-game.kuki

import "stdlib/game"

function main()
    _ = game.Window("Hello Kukicha!", 640, 480)
        |> game.OnDraw(draw)
        |> game.Run() onerr panic "{error}"

function draw(screen game.Screen)
    game.Clear(screen, game.MakeColor(30, 30, 60, 255))
    game.DrawText(screen, "Hello, Kukicha!", 240, 220, game.White)
```

## How it works

1. `game.Window("Hello Kukicha!", 640, 480)` creates a 640x480 window with a title.
2. `|> game.OnDraw(draw)` tells the game to call our `draw` function every frame.
3. `|> game.Run()` starts the game loop. The `onerr panic` handles any startup errors.
4. Inside `draw`, we clear the screen to a dark blue, then draw white text.

## Build and run

```bash
kukicha build --wasm hello-game.kuki
# Produces: hello-game.wasm, wasm_exec.js, index.html
python3 -m http.server
# Open http://localhost:8000 in your browser
```

## Try it

- Change the color values in `MakeColor` (each is 0-255 for red, green, blue, alpha)
- Change the text position (240, 220) to move the text around
- Try `game.Black`, `game.Red`, or `game.Blue` instead of `game.White`

## Key concepts

| Concept | What it does |
|---------|-------------|
| `import "stdlib/game"` | Load the game library |
| `function` | Define a named function (beginner alias for `func`) |
| `\|>` (pipe) | Pass the result of one function to the next |
| `onerr panic` | If something goes wrong, crash with a message |
