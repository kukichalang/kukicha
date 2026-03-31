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
| Compression | Pre-compressed Brotli (`.wasm.br` in Docker) + `Content-Encoding: br` | Self-contained, no CDN needed for Brotli |

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

### v0.2 — Playground (transpiler-only WASM)

#### Architecture

The `compile()` pipeline in `cmd/kukicha/main.go` is entirely pure Go — no `exec.Command`.
Shell-outs only happen in `buildCommand` (calls `go build`) and `runCommand` (calls `go run`).

This means **parse → semantic → codegen → `go/format`** can run in WASM with no backend and
no Go toolchain in the browser. The compiler internals need zero changes.

A new `cmd/kukicha-wasm/main.go` (~50 lines of glue) exposes a single JS function:

```
kukichaTranspile(source) → { goSource: "...", errors: ["..."] }
```

Estimated WASM size: ~3–5 MB uncompressed, ~1–2 MB gzipped.

**What the playground does:** Kukicha source on the left → live-generated Go on the right.
Errors appear as you type (debounced 300ms). No execution — transpilation + error display only.
This directly serves selling points 1 ("read what AI writes") and 2 ("catch mistakes at compile time").

**Execution (actually running Kukicha code) is deferred to v0.3** — it requires a sandboxed
server endpoint calling `go run`, rate limiting, and streaming stdout/stderr back.

#### Checklist
- [x] `cmd/kukicha-wasm/main.go` — WASM entrypoint, registers `kukichaTranspile` JS function
- [x] `kukicha build --wasm cmd/kukicha-wasm/main.kuki` → `static/wasm/kukicha.wasm`
- [x] Split-pane editor (CodeMirror or plain `<textarea>` to start)
- [x] Live transpilation on keystroke (debounced 300ms, plain JS — no HTMX needed here)
- [x] Pre-loaded examples: fetch pipeline, hello world, security flag demo
- [x] Error display below editor pane
- [x] "Generated Go" pane collapses on mobile
- [x] HTMX partial: swap playground section without full page reload

### v0.3 — Playground execution + Docs migration

#### Playground execution

**Architecture:** Single `POST /api/run` endpoint. Receives `{"source": "..."}` (max 64 KB), returns `Content-Type: text/event-stream`. HTMX 4's SSE extension auto-detects the stream — no two-step flow needed (HTMX 4 uses `fetch()` instead of `EventSource`, so POST → SSE works natively). Unnamed SSE messages carry HTML fragments (stdout/stderr lines) and are auto-swapped by HTMX into the output pane. Named `exit` event carries `{"code", "duration_ms"}` and is handled by a small JS listener.

**Sandbox:** nsjail (Google's lightweight process isolator). Runs `kukicha run --playground temp.kuki` in an isolated namespace with no network, read-only filesystem (except `/tmp` tmpfs), 10s timeout, 256 MB memory, 32 max processes. Go module cache pre-warmed at Docker build time and bind-mounted read-only (`GOPROXY=off`).

**Defense-in-depth:** `--playground` flag on `kukicha run` (change in kukichalang/kukicha repo) rejects imports of `os/exec`, `net`, `net/http`, `syscall`, `unsafe`, `plugin` at compile time with clear error messages — before nsjail even gets involved.

##### Server & sandbox
- [ ] `handlers/run.kuki` — POST /api/run: validate body (64 KB max), rate-limit, write temp .kuki file, spawn nsjail, stream SSE
- [ ] `nsjail.cfg` — nsjail protobuf config (ONCE mode, 10s timeout, 256 MB mem, no network, read-only mounts, `CGO_ENABLED=0`)
- [ ] `scripts/warmup.sh` — Pre-warm Go module cache during Docker build (stdlib transitive deps)
- [ ] `Dockerfile` updates — Add nsjail, Go toolchain, kukicha binary, module cache warmup
- [ ] `--playground` flag on `kukicha run` (kukichalang/kukicha repo) — import blocklist for dangerous packages

##### Rate limiting & concurrency
- [ ] `handlers/ratelimit.kuki` — In-memory token bucket (10 req/min per IP, 50 req/hr per IP)
- [ ] Global concurrency semaphore — Max 5 concurrent sandbox executions, 503 if at capacity

##### HTMX 4 frontend
- [ ] "Run" button with `hx-post="/api/run"` + `hx-vals='js:{"source": editor.getValue()}'` + `hx-swap="beforeend"` + `hx-target="#output-pane"`
- [ ] Output pane: unnamed SSE messages (HTML fragments) auto-swapped by HTMX 4
- [ ] `static/js/playground.js` — JS listener for named `exit` event → status bar update
- [ ] `<meta name="htmx-config" content='{"extensions": "sse"}'>` to activate SSE extension

#### Docs migration
- [ ] Move tutorials into the website (render markdown → `html.Fragment` at build time or startup)
- [ ] Stdlib reference pages (auto-generated from .kuki files)
- [ ] Search (HTMX `hx-trigger='keyup changed delay:300ms'` for live search)

### Cloud CDN (long-term)
- [ ] Global HTTPS Load Balancer in front of Cloud Run (GCP)
- [ ] Enable Cloud CDN on the LB backend for edge caching of WASM + static assets
- [ ] Add `Vary: Accept-Encoding` already in place — CDN will cache Brotli and plain versions separately
- [ ] Expected savings: WASM files served from edge, ~300ms → <50ms for cached requests; eliminates Cloud Run cold starts for static asset fetches

Note: The `Vary: Accept-Encoding` and `Cache-Control: public, max-age=3600` headers are already set by the `serveWasm` handler in `main.kuki`, so the LB/CDN setup requires no application code changes.

### v0.4 — Blog
- [ ] Markdown → `html.Fragment` renderer
- [ ] Blog index page with HTMX pagination
- [ ] First post: "Why we built Kukicha"
- [ ] RSS feed (plain `html.Render()` with XML content type)
