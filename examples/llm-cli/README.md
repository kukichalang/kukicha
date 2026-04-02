# owui — CLI agent: Open WebUI + Open Terminal via MCP

A terminal-native agent that gets its brain from [Open WebUI](https://github.com/open-webui/open-webui) (LLM) and its hands from [Open Terminal](https://github.com/open-webui/open-terminal) (sandboxed shell), connected via the [Model Context Protocol](https://modelcontextprotocol.io).

```
┌─────────┐     tools (from MCP)       ┌───────────────┐
│         │ ──── chat/completions ────→ │  Open WebUI    │
│  owui   │ ←─── stream + tool_calls ── │  (LLM)         │
│  (CLI)  │                             └───────────────┘
│         │     MCP (streamable-http)   ┌───────────────┐
│         │ ──── CallTool ────────────→ │ Open Terminal  │
│         │ ←─── results ────────────── │ (MCP server)   │
└─────────┘                             └───────────────┘
```

**Tools are discovered dynamically.** At startup, owui connects to Open Terminal's
MCP server and calls `ListTools` — whatever endpoints your instance exposes
(execute, files, search, terminals, notebooks…) become available to the model
automatically. No hardcoded tool definitions, no version drift.

Built with [Kukicha](https://github.com/kukichalang/kukicha) and its standard library:
- `stdlib/llm` — OpenAI-compatible chat completions (streaming, tool calls)
- `stdlib/mcp` — MCP client (connect, list tools, call tools)
- `stdlib/fetch` — HTTP requests
- `stdlib/input` — readline prompts
- `stdlib/json` — JSON marshalling

## Install

```bash
kukicha build examples/llm-cli/
# binary at ./owui
```

## Configure

```bash
# Environment
export OWUI_WEBUI_URL="http://localhost:3000"
export OWUI_WEBUI_API_KEY="sk-..."
export OWUI_MODEL="llama3.1"
export OWUI_TERMINAL_MCP_URL="http://localhost:8000/mcp"

# Or interactive
owui configure

# Verify
owui health
owui tools    # see what the MCP server exposes
owui models   # see what LLMs are available
```

## Usage

### Agent mode (LLM + MCP tools)

```bash
# One-shot
owui "set up a python project with fastapi and write a hello world"

# Pipe context
cat error.log | owui "diagnose and fix this"

# Raw output for piping
owui "list installed python packages" --raw | grep torch

# Interactive chat
owui -c

# Seed the system prompt
owui -c -S "you are a senior Go developer. be terse."

# Pipe context then chat
cat main.go | owui -c "review this code"
```

Tool calls show on stderr:
```
MCP connected: 12 tools from http://localhost:8000/mcp
-> execute_command {"command":"ls -la /workspace"}
<- execute_command total 24 drwxr-xr-x 3 user user 4096...
-> write_file {"path":"/workspace/main.py","content":"from fa...
<- write_file Wrote 342 bytes to /workspace/main.py
The FastAPI project is set up...
(3 tool rounds)
```

### Inspect

```bash
owui tools     # list all tools discovered from MCP
owui models    # list available LLMs
owui health    # check connectivity to both services
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--chat` | `-c` | Interactive chat (readline loop) |
| `--model` | `-m` | Override model |
| `--system` | `-S` | Override system prompt |
| `--raw` | | Raw output, no formatting |

## Architecture

```
main.kuki        CLI entrypoint, flag parsing, subcommands, interactive chat
config.kuki      Config loading (env > file > defaults)
agent.kuki       Agent loop: LLM → tool calls → MCP dispatch → repeat
bridge.kuki      MCP client → discovers tools, converts to stdlib/llm
                 format, dispatches CallTool via MCP protocol
```

The key file is `bridge.kuki`:

1. **Connect** — `mcp.Connect` to Open Terminal via streamable HTTP
2. **Discover** — `mcp.ListTools` gets all available tools with schemas
3. **Convert** — MCP tool schemas → `llm.Tool` format for the LLM
4. **Dispatch** — `mcp.CallTool` executes tools and returns results

No hardcoded tool definitions anywhere. If Open Terminal adds a new endpoint,
your CLI picks it up on the next run.

## Prerequisites

- Open WebUI running (for the LLM)
- Open Terminal running with MCP enabled:
  ```bash
  pip install open-terminal[mcp]
  open-terminal mcp --transport streamable-http
  ```
  Or the standard HTTP server works too — `FastMCP.from_fastapi` exposes `/mcp`
  alongside the REST API.

## License

MIT
