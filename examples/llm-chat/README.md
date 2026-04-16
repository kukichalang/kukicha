# LLM Chat

End-to-end demo of `stdlib/llm` across the three API formats it
supports:

- **Chat Completions** (OpenAI-compatible)
- **OpenResponses** (<https://www.openresponses.org>)
- **Anthropic Messages** (<https://docs.anthropic.com/en/api/messages>)

Each section covers one-shot completions, multi-turn conversations,
and streaming.

## Prerequisites

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

Or, to route everything through a gateway like `any-llm-gateway`:

```bash
export LLM_API_KEY="your-gateway-key"
```

## Running

```bash
kukicha run examples/llm-chat/main.kuki
```
