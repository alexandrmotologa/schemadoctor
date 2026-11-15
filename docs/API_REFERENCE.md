# API and CLI Reference

## CLI Commands

### `schemadoctor serve`

Starts the reverse proxy and embedded React 19 studio dashboard.

```bash
schemadoctor serve [flags]
```

Flags:
- `--port int`: Reverse proxy port (default: `8080`).
- `--studio-port int`: Management API and React studio port (default: `4141`).
- `--upstream string`: Upstream LLM provider base URL (default: `https://api.openai.com`).
- `--mock-upstream`: Starts an integrated simulated upstream LLM server for local offline testing.
- `--config string`: Path to YAML configuration file.
- `--db string`: Path to SQLite database file (default: `data/schemadoctor.db`).
- `--schemas string`: Directory path containing `*.json` schema files (default: `schemas`).

### `schemadoctor fix`

Repairs malformed JSON from standard input or a file.

```bash
# From file
schemadoctor fix path/to/broken.json

# From standard input
cat broken.json | schemadoctor fix

# With schema type coercion
schemadoctor fix --schema schemas/user_profile.json broken.json

# Overwrite file in place
schemadoctor fix --in-place broken.json
```

Flags:
- `--schema string`: Name of a registered schema or path to a JSON Schema file.
- `--indent bool`: Format output with two-space indentation (default: `true`).
- `--in-place bool`: Overwrite target file with repaired output (default: `false`).

### `schemadoctor bench`

Runs internal benchmarks measuring throughput and latency across syntax failure patterns.

```bash
schemadoctor bench -n 50000
```

---

## Reverse Proxy Endpoints

### `POST /v1/chat/completions`

Proxies requests to the upstream provider while inspecting and healing response JSON payloads.

Request Headers:
- `Authorization: Bearer <key>`: Passed transparently to upstream.
- `X-SchemaDoctor-Target-Schema: <name>`: Optional target schema name for type coercion.
- `X-SchemaDoctor-Upstream: <url>`: Dynamic upstream host override for this specific request.

Path-Based Multi-Provider Routing:
- `/openai/*`: Forwards directly to `https://api.openai.com` (e.g. `/openai/v1/chat/completions`).
- `/anthropic/*`: Forwards directly to `https://api.anthropic.com` (e.g. `/anthropic/v1/messages`).
- `/ollama/*`: Forwards directly to `http://localhost:11434` (e.g. `/ollama/api/chat`).

Supported Payload Locations:
- Standard chat completions: `choices[0].message.content`
- Function calling and tool use: `choices[0].message.tool_calls[i].function.arguments`
- Streaming tool calls: `choices[0].delta.tool_calls[i].function.arguments`
- Anthropic message blocks: `content[i].text` and `content[i].input` (for `type: "tool_use"`)

Response Headers added by schemadoctor:
- `X-SchemaDoctor-Healed: true`: Present when syntax repairs were applied.
- `X-SchemaDoctor-Location: <content|tool_arguments|anthropic_text|anthropic_tool_input>`: Indicates where the repair was applied.
- `X-SchemaDoctor-Latency-Us: <int>`: Repair processing latency in microseconds.

---

## Management and Studio REST API

The management server listens on `:4141` by default.

### `GET /health`

Health check endpoint.

Response:
```json
{
  "status": "ok",
  "uptime_seconds": 128.4,
  "version": "1.0.0"
}
```

### `GET /metrics`

Standard Prometheus text exposition format endpoint for scraping metrics with Prometheus and Grafana.

Example output:
```
# HELP schemadoctor_requests_total Total number of requests intercepted by schemadoctor proxy
# TYPE schemadoctor_requests_total counter
schemadoctor_requests_total 1420

# HELP schemadoctor_repaired_requests_total Total number of requests healed by schemadoctor
# TYPE schemadoctor_repaired_requests_total counter
schemadoctor_repaired_requests_total 380

# HELP schemadoctor_tokens_saved_total Total estimated LLM tokens saved by avoiding API retries
# TYPE schemadoctor_tokens_saved_total counter
schemadoctor_tokens_saved_total 133000

# HELP schemadoctor_repair_latency_microseconds Average latency of the AST repair engine in microseconds
# TYPE schemadoctor_repair_latency_microseconds gauge
schemadoctor_repair_latency_microseconds 2.650

# HELP schemadoctor_errors_healed_total Cumulative count of syntax errors repaired by error category
# TYPE schemadoctor_errors_healed_total counter
schemadoctor_errors_healed_total{type="truncation"} 195
schemadoctor_errors_healed_total{type="markdown"} 110
schemadoctor_errors_healed_total{type="quotes"} 45
schemadoctor_errors_healed_total{type="trailing_comma"} 80
schemadoctor_errors_healed_total{type="dangling_key"} 18
schemadoctor_errors_healed_total{type="type_coercion"} 42
```

### `GET /api/stats`

Returns cumulative metrics and error category counters.

Response:
```json
{
  "total_requests": 1420,
  "repaired_requests": 380,
  "valid_pass_through": 1040,
  "failed_repairs": 0,
  "success_rate": 100.0,
  "tokens_saved_estimated": 133000,
  "avg_latency_us": 2.65,
  "truncation_fixed": 195,
  "markdown_stripped": 110,
  "quotes_fixed": 45,
  "trailing_commas_fixed": 80,
  "dangling_keys_fixed": 18,
  "type_coercions": 42
}
```

### `GET /api/logs`

Returns paginated telemetry events.

Query Parameters:
- `limit int`: Maximum items to return (default: `50`, max: `200`).
- `offset int`: Offset index (default: `0`).

Response:
```json
{
  "events": [
    {
      "id": 1,
      "trace_id": "trc-a81f3b20",
      "timestamp": "2026-11-15T19:30:00Z",
      "provider": "openai",
      "model": "gpt-4o-mini",
      "status": "healed",
      "latency_us": 240,
      "tokens_saved": 420,
      "original_size": 180,
      "repaired_size": 175,
      "flags": "stripped_markdown, balanced_delimiters",
      "raw_input": "```json\n{\"id\": 101, \"tags\": [\"ai\", \"llm\", ]}\n```",
      "repaired_output": "{\"id\": 101, \"tags\": [\"ai\", \"llm\"]}"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

### `POST /api/repair`

Manual test and playground repair endpoint.

Request:
```json
{
  "input": "{\"user\": \"Alex\", \"items\": [1, 2, ",
  "schema": "user_profile"
}
```

Response:
```json
{
  "valid": true,
  "repaired": "{\"user\":\"Alex\",\"items\":[1,2]}",
  "flags": {
    "stripped_markdown": false,
    "closed_open_string": false,
    "balanced_delimiters": true,
    "removed_trailing_comma": true,
    "completed_dangling_key": false,
    "normalized_ident": false,
    "fixed_unescaped_quote": false
  },
  "coerced_count": 0,
  "latency_us": 18,
  "tokens_saved": 110
}
```

### `GET /api/schemas`

Lists all registered JSON Schemas.

### `POST /api/schemas`

Registers a new JSON Schema at runtime.

Request:
```json
{
  "name": "order_payload",
  "schema": {
    "type": "object",
    "properties": {
      "order_id": {"type": "string"},
      "total": {"type": "number"},
      "paid": {"type": "boolean"}
    },
    "required": ["order_id", "total"]
  }
}
```

### `POST /api/schemas/infer`

Infers a standard JSON Schema from an example JSON payload.

Request:
```json
{
  "input": "{\"user_id\": 101, \"username\": \"alex\", \"roles\": [\"admin\"]}",
  "title": "UserAccount"
}
```

Response:
```json
{
  "title": "UserAccount",
  "schema": {
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "title": "UserAccount",
    "type": "object",
    "properties": {
      "roles": {
        "type": "array",
        "items": {"type": "string"}
      },
      "user_id": {"type": "integer"},
      "username": {"type": "string"}
    },
    "required": ["roles", "user_id", "username"]
  }
}
```

### `POST /api/simulate`

Simulates a realistic batch of 5 malformed LLM responses through the proxy telemetry engine.

Response:
```json
{
  "simulated_count": 5,
  "status": "completed",
  "message": "Successfully sent 5 simulated malformed requests through proxy telemetry pipeline"
}
```

### `GET /api/stream`

Server-Sent Events endpoint broadcasting live telemetry events to connected dashboards in real time.
