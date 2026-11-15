# Architecture and System Design

## Overview

schemadoctor is a reverse proxy and AST repair engine designed to repair malformed or truncated JSON from large language models on the fly. It runs as a single binary in Go, consuming less than 15MB of RAM and adding less than 1ms of latency to upstream HTTP and SSE requests.

```
 Client Application (LangChain, OpenAI SDK, LlamaIndex)
                     │
                     │ HTTP / SSE (/v1/chat/completions)
                     ▼
       ┌───────────────────────────────┐
       │   schemadoctor Reverse Proxy  │
       │   Port 8080                   │
       └──────────────┬────────────────┘
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
[Buffered Mode]            [Streaming SSE Mode]
Reads response body        Intercepts token deltas
Tokenizes AST              Tracks delimiter stack
Applies repair engine      Injects closing tokens before [DONE]
Coerces schema types       Broadcasts SSE telemetry
Updates Content-Length     Forwards clean tokens to client
        │                           │
        └─────────────┬─────────────┘
                      ▼
         Upstream LLM Provider / Local Model
         (OpenAI, Anthropic, Ollama, vLLM)
```

## Core Subsystems

### 1. Incremental AST Lexer and Sanitizer (`internal/ast`)

The lexer scans incoming bytes without heap allocations. It identifies structural delimiters (`{`, `}`, `[`, `]`, `:`, `,`), string literals, numbers, booleans, and nulls.

The sanitizer handles three main categories of text defects:
- Markdown fences: drops leading conversational text and fences (` ```json `) up to the first `{` or `[`.
- Trailing commentary: strips closing fences and conversational epilogues after the root delimiter closes.
- Unescaped nested quotes: detects unescaped double quotes inside string values by inspecting the succeeding token.

### 2. Delimiter Stack Balancer (`internal/ast/stack.go` and `repair.go`)

A LIFO delimiter stack records every open `{` and `[`.

When an LLM response is truncated due to `max_tokens` or early stream termination:
1. If the scanner was inside an unterminated string literal, it appends a closing double quote.
2. If the last token was an unfinished key followed by a colon (`"key":`), it inserts `null`.
3. If the last token was a trailing comma, it trims the comma.
4. It pops open delimiters from the stack in reverse order and appends the matching closing brackets and braces (`]` and `}`).

For streaming Server-Sent Events, `CalculateClosingSuffix` scans the accumulated tokens and injects the necessary closing characters inside a synthetic final delta chunk right before `data: [DONE]`.

### 3. Schema Registry and Type Coercer (`internal/schema`)

The schema engine compiles JSON Schema definitions into memory using `santhosh-tekuri/jsonschema/v5`.

When a client specifies a target schema via `X-SchemaDoctor-Target-Schema`:
- String numbers (`"42"`, `"98.5"`) are parsed into numeric integers or floats when the schema expects numbers.
- String booleans (`"true"`, `"false"`, `"yes"`, `"no"`) are coerced into booleans.
- Primitive singletons (`"tag"`) are wrapped into arrays (`["tag"]`) if the schema requires an array.
- Empty strings are converted to null when nullable.

### 4. Telemetry and Zero-CGO SQLite (`internal/telemetry`)

Telemetry records every repair event:
- Trace ID, model, provider, status (healed, valid, failed).
- Latency in microseconds.
- Estimated tokens saved by avoiding an LLM error retry.
- Bitmask flags of applied repairs.

The store uses `modernc.org/sqlite`, a pure Go SQLite engine requiring zero CGO. An in-memory event bus broadcasts new repair events to connected Web UI clients through Server-Sent Events (`/api/stream`).

### 5. Embedded Studio (`internal/server` and `web/`)

The web studio is built with React 19, TypeScript, and Tailwind CSS. The compiled production bundle is embedded directly inside the Go binary via `//go:embed all:dist` in `web/embed.go`.

The studio provides:
- Live monitoring dashboard with request counts, success rates, and token savings.
- Real-time request log with latency measurements and applied repair tags.
- Side-by-side colorized diff viewer comparing raw input to healed output.
- Interactive playground with preloaded LLM failure presets.
- Schema registry manager to inspect and register JSON schemas dynamically.
