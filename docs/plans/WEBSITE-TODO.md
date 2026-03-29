# kukicha.org Website — TODO

New repo: `kukichalang/kukicha.org`

## How Other Languages Do It

Open-source language websites to learn from:

| Language | Repo | Stack | Notes |
|----------|------|-------|-------|
| Go | [golang/website](https://github.com/golang/website) | Go + HTML templates | Serves go.dev, includes playground backend |
| Rust | [rust-lang/www.rust-lang.org](https://github.com/rust-lang/www.rust-lang.org) | Rocket (Rust) + Handlebars | Translations, governance pages |
| Zig | [ziglang/www.ziglang.org](https://github.com/ziglang/www.ziglang.org) | Hugo (static) | Docs + blog, CI deploys |
| Gleam | [gleam-lang/website](https://github.com/gleam-lang/website) | Static (Gleam-generated) | Clean, personality-forward |
| Dart | [dart-lang/site-www](https://github.com/dart-lang/site-www) | Jekyll | Docs-heavy, playground embedded |
| Swift | [swiftlang/swift-org-website](https://github.com/swiftlang/swift-org-website) | Jekyll | Blog + docs + getting started |

Common patterns:
- Static site generators (Hugo, Jekyll) are the norm
- Playground is either embedded iframe or separate service
- All accept community PRs for docs/tutorials
- Blog/changelog lives in the same repo

## Stack

**Built in Kukicha.** The website is the showcase — no Hugo, no Jekyll, no static site generator. The site is a Kukicha binary serving `stdlib/html` fragments with HTMX 4 for interactivity and Oat CSS for styling.

| Layer | Choice | Why |
|-------|--------|-----|
| Server | Kukicha (`stdlib/http`) | Dog-food the language |
| HTML | `stdlib/html` (Fragment components) | Dog-food the stdlib |
| Interactivity | HTMX 4 (CDN) | No JS build step, partial swaps for WASM loader |
| CSS | Oat (CDN) | Classless + utility, no build step |
| WASM demos | Pre-built `.wasm` in `static/` | Stem Panic, Breakout, playground (future) |

This means:
- The entire site source is `.kuki` files — visitors can read the code that serves them
- Components are functions: `Navbar()`, `Hero()`, `Card()`, `CodeCompare()`
- HTMX partials are just functions returning `html.Fragment` — same function serves full-page and partial requests
- Security comes free: `html.Escape()` for user input, compiler XSS checks
- Deploys as a single binary + static assets (WASM files, `wasm_exec.js`)

Reference implementation: `examples/homepage/main.kuki`

Hosting: Fly.io, Railway, or any container host (single binary, <10MB). Cloudflare Pages won't work for a server — use Cloudflare as CDN/DNS in front if desired.

## Credits

### Ebitengine
The game demos and game tutorial series are built with [Ebitengine](https://ebitengine.org/) (by Hajime Hoshi) — a production-grade 2D game engine for Go. Kukicha's `game` stdlib package ([kukichalang/game](https://github.com/kukichalang/game)) wraps Ebitengine to provide a beginner-friendly API.

### templ
Kukicha's `stdlib/html` package is inspired by [templ](https://templ.guide/) (by Adrian Hesketh) — a type-safe HTML templating language for Go. templ's component model (typed fragments, composition via nesting, auto-escaping at the boundary) directly informed the design of `html.Fragment`, `html.Escape()`, and `html.Embed()`. The key difference: templ uses code generation from `.templ` files, while `stdlib/html` uses Kukicha's native string interpolation at runtime.

The website must include visible credit:
- Footer: "Games powered by [Ebitengine](https://ebitengine.org/)" and "HTML inspired by [templ](https://templ.guide/)"
- Game demo page: "Built with Kukicha's game library, which wraps [Ebitengine](https://ebitengine.org/) by Hajime Hoshi"
- License compliance: Ebitengine is Apache 2.0 — include in NOTICE file (templ credit is courtesy, not a license requirement — no templ code is used)

## Landing Page Structure

### Hero Section
Playable Stem Panic running in WASM. No install, no signup, just play.

Headline: "A game that teaches you a programming language, written in that programming language."

Subhead: "Kukicha compiles to Go. Ship as a single binary, WASM, or anywhere Go runs."

### Source Reveal
Collapsible syntax-highlighted source below the game. "810 lines. No framework."

Link to the 8-lesson game tutorial series that builds up to it.

### Interactive Playground
Split-pane editor, pre-loaded with the fetch + filter + sort pipeline example.
Runs via WASM in-browser. No backend needed for basic examples.

### Three Selling Points

1. **"Read what AI writes"** — Kukicha is designed for humans to review AI-generated code. English keywords, no symbol soup, 1:1 mapping to Go and Python concepts.

2. **"Catch AI mistakes at compile time"** — The compiler flags SQL injection, XSS, SSRF, path traversal, command injection, and open redirects before your code ships.

3. **"Ship anywhere Go runs"** — Single binary. No runtime. WASM for the browser. Goroutines for concurrency.

### Tutorial Funnel

Two paths:

**"I want to make something"**
- Game tutorials (8 lessons, Breakout in the browser)
- CLI tool tutorial
- Web app tutorial
- AI/MCP agent tutorial

**"I want to understand the language"**
- Absolute beginner tutorial
- Quick reference (Go/Python translation table)
- Stdlib reference

### Footer
- GitHub: kukichalang/kukicha
- Editor support: VS Code, Zed
- Games powered by [Ebitengine](https://ebitengine.org/)
- Version: current release

## Repo Structure

```
kukichalang/kukicha.org/
  main.kuki               # Entry point — routes, server startup
  components/
    layout.kuki           # Layout(), Head(), Navbar(), Footer()
    hero.kuki             # Hero(), WasmDemo(), WasmPlayer()
    cards.kuki            # Card(), CardGrid(), Feature type
    code.kuki             # CodeCompare(), SourceReveal()
    tutorials.kuki        # TutorialFunnel()
  pages/
    home.kuki             # HomePage() — composes all components
    playground.kuki       # PlaygroundPage() (future)
    docs.kuki             # DocsPage() (future)
  static/
    wasm/                 # Pre-built WASM binaries (Stem Panic, Breakout)
    wasm_exec.js          # Go WASM support file
  Dockerfile              # FROM scratch, COPY binary + static/
  NOTICE                  # Third-party credits (Ebitengine, Oat, HTMX)
  README.md
```

No `go.mod` in this repo — it's a Kukicha project. `kukicha build main.kuki` produces the binary.

## WASM Build Pipeline

The game demos need a CI step:
1. `kukicha build --wasm examples/stem-panic/main.kuki`
2. `kukicha build --wasm examples/breakout/breakout.kuki` (extract from tutorial 08)
3. Copy `.wasm` binaries + Go's `wasm_exec.js` to `static/wasm/`
4. HTMX lazy-loads the game: click "Load Game" → `hx-get='/wasm-player'` returns a `<canvas>` + `<script>` fragment

The playground needs a WASM-compiled Kukicha compiler (heavier lift, defer to v0.2).

## How stdlib/html Drives the Architecture

Each page and component is a Kukicha function returning `html.Fragment`:

```kukicha
# pages/home.kuki
func HomePage() html.Fragment
    return html.Join(
        Navbar("kukicha", navLinks),
        Hero("A language you can read", ...),
        CardGrid(features),
        CodeCompare(),
        WasmDemo(),
        TutorialFunnel(),
        Footer(),
    )
```

HTMX partials are the same functions, served from separate routes:

```kukicha
# Full page: GET /
page := Layout("Kukicha", HomePage())
html.WriteTo(w, page) onerr discard

# HTMX partial: GET /wasm-player (returns just the canvas fragment)
html.WriteTo(w, WasmPlayer()) onerr discard
```

Key patterns:
- `html.Escape()` for all user/dynamic content (XSS safe by convention)
- `html.Embed()` for composing child fragments (no double-escaping)
- `html.When()` / `html.WhenElse()` for conditional sections
- `html.Join()` for stacking page sections
- Single-quoted HTML attributes (`class='card'`) — no backslash escaping needed
- Oat CSS via CDN `<link>` — no build step, classless defaults + utility classes
- HTMX via CDN `<script>` — `hx-get`, `hx-target`, `hx-swap='morph'` for partials

## Milestones

### v0.1 — Landing page (Kukicha-powered)
- [ ] Create `kukichalang/kukicha.org` repo
- [ ] Port `examples/homepage/main.kuki` into repo structure (split into components/)
- [ ] Build Stem Panic WASM binary, add to `static/wasm/`
- [ ] HTMX lazy-load for game (click to play, not auto-load)
- [ ] Source reveal section (collapsible `<details>` with syntax-highlighted Kukicha)
- [ ] Three selling points (cards)
- [ ] Code comparison (Go vs Kukicha side-by-side)
- [ ] Tutorial links (point back to main repo docs for now)
- [ ] Ebitengine credit in footer and game section
- [ ] Dockerfile (FROM scratch, single binary + static/)
- [ ] Deploy to Fly.io or Railway
- [ ] Cloudflare DNS pointing kukicha.org at the host

### v0.2 — Playground
- [ ] WASM-compiled Kukicha compiler running in browser
- [ ] Split-pane editor with syntax highlighting
- [ ] Pre-loaded examples (fetch pipeline, Breakout, hello world)
- [ ] HTMX partial: swap playground output without full page reload

### v0.3 — Docs migration
- [ ] Move tutorials into the website (render markdown → `html.Fragment` at build time or startup)
- [ ] Stdlib reference pages (auto-generated from .kuki files)
- [ ] Search (HTMX `hx-trigger='keyup changed delay:300ms'` for live search)

### v0.4 — Blog
- [ ] Markdown → `html.Fragment` renderer
- [ ] Blog index page with HTMX pagination
- [ ] First post: "Why we built Kukicha"
- [ ] RSS feed (plain `html.Render()` with XML content type)
