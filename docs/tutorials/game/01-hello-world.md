# Lesson 1: Hello World

Your first Kukicha game — a window with colored text.

## What you'll learn

- `import` to load the game library
- `function` to define your own functions
- Pipes (`|>`) to chain function calls
- `game.Window`, `game.OnDraw`, `game.Clear`, `game.DrawText`, `game.Run`

## Project setup

Create a new directory and initialize a Kukicha project:

```bash
mkdir hello-game && cd hello-game
kukicha init
```

This extracts the standard library and sets up your `go.mod`. Since Kukicha uses Go's package system under the hood, you can install any Go package with `go get`. The game library lives in its own module, so add it:

```bash
go get github.com/kukichalang/game@latest
```

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
```

The browser can't load `.wasm` files from the filesystem directly — you need a
small HTTP server. Create `serve.kuki` in the same directory:

> **Kukicha vs Go imports:** Imports starting with `stdlib/` are Kukicha's
> standard library — designed to be beginner-friendly. Imports like `"net/http"`
> and `"fmt"` come from Go's standard library. Since Kukicha compiles to Go,
> you can use any Go package directly. You'll mostly use `stdlib/` packages,
> but occasionally Go packages are handy when Kukicha doesn't wrap them yet.

```kukicha
# serve.kuki — a tiny web server to run your game in the browser

import "net/http"
import "mime"
import "fmt"

function main()
    # Register the WebAssembly MIME type so browsers accept .wasm files
    mime.AddExtensionType(".wasm", "application/wasm") onerr panic "{error}"

    # Print the address so you know where to go
    fmt.Println("Open http://localhost:8080 in your browser")

    # Serve every file in the current folder on port 8080
    # This lets the browser load your .wasm, wasm_exec.js, and index.html
    http.ListenAndServe(":8080", http.FileServer(http.Dir("."))) onerr panic "{error}"
```

Then:

```bash
kukicha run serve.kuki
# Open http://localhost:8080 in your browser
```

> **Tip:** You only need one `serve.kuki` — reuse it for every tutorial.

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
