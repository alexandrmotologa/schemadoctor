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

Response Headers added by schemadoctor:
- `X-SchemaDoctor-Healed: true`: Present when syntax repairs were applied.
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

### `GET /api/stream`

Server-Sent Events endpoint broadcasting live telemetry events to connected dashboards in real time.
