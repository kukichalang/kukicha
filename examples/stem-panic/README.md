# Stem Panic

A tapper-style tea processing game built with Kukicha and the `stdlib/game` library.

Slide raw tea stems down conveyor lanes to fill tea orders. Each lane is a `|>` pipeline with processing stations inline. An unhandled bitter leaf causes a Bitter Panic!

## How to play

- **UP/DOWN** — switch lanes
- **ENTER/SPACE** — send a stem down the current lane
- **SPACE** — activate the `onerr` catcher to catch bitter leaves
- **ESC** — return to title screen

## Building

This is a WASM-only game. Build and serve it in the browser:

```bash
kukicha build --wasm main.kuki
kukicha run serve.kuki
# Open http://localhost:8082
```

## Concepts demonstrated

This example builds on the [game tutorials](../../docs/tutorials/game/01-hello-world.md) and adds:

- Multiple game states (title, menu, playing, level complete, game over)
- Level progression with unlockable stages
- Multiple conveyor lanes with independent processing stations
- The `|>` pipe operator as a visual gameplay mechanic
- `onerr` catchers as interactive game objects
- Enums for game state management

## Files

| File | Purpose |
|------|---------|
| `main.kuki` | Game logic (WASM, `//go:build js`) |
| `serve.kuki` | Local dev server (native Go) |
