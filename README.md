<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="schemadoctor Logo" width="130" style="border-radius: 24px;" />
</p>
<h1 align="center">schemadoctor</h1>

<p align="center">
  <strong>Zero-latency streaming JSON self-healing reverse proxy and incremental AST repair engine for LLMs and local SLMs.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/React-19.0-61DAFB?style=flat-square&logo=react" alt="React 19" />
  <img src="https://img.shields.io/badge/SQLite-Pure--Go-003B57?style=flat-square&logo=sqlite" alt="SQLite" />
  <img src="https://img.shields.io/badge/Wire%20Latency-%3C%200.004ms-brightgreen?style=flat-square" alt="Wire Latency" />
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="License" />
</p>

<p align="center">
  <img src="docs/images/schemadoctor_demo.gif?raw=true" alt="schemadoctor Real-Time Interception and Self-Healing Demo" width="100%" />
</p>

schemadoctor intercepts HTTP and Server-Sent Events (SSE) responses from language model APIs (OpenAI, Anthropic, Ollama, vLLM) and repairs malformed, truncated, or invalid JSON on the wire in sub-millisecond time. It prevents client parser crashes and eliminates expensive API retry calls.

## Problem

Structured JSON generation is necessary for tool calling, agentic workflows, and backend data ingestion. However, generation frequently breaks in production environments:

1. Token limit truncation: When generation reaches `max_tokens`, outputs cut off mid-string or mid-object, leaving unclosed brackets (`[`), braces (`{`), and strings.
2. Markdown fences: Models frequently wrap outputs in ````json ... ```` fences or prepend conversational chatter ("Here is your JSON:").
3. Unescaped quotes: Models emit raw double quotes inside nested string values, invalidating JSON decoders.
4. Trailing commas: Models produce trailing commas before closing delimiters (`{"items": [1, 2, ],}`).
5. Dangling keys: Generation terminates after emitting a key and colon without a value (`{"status": }`).
6. Python literals: Models emit `True`, `False`, or `None` instead of standard JSON literals.
7. Type collisions: Models return numbers or booleans encoded as strings (such as `"age": "28"`), failing strict schema validation.

Standard remedies rely on retrying the prompt ("Fix your previous JSON"), which consumes two to three times the token budget and adds seconds of latency. Grammar-constrained decoding slows generation throughput and is not supported by closed-source provider APIs.

schemadoctor repairs these syntax defects deterministically in under 3 microseconds without consuming extra tokens.

## Key Capabilities

- Streaming SSE balance: Intercepts Server-Sent Events token chunks and injects closing delimiters before `data: [DONE]`.
- Tool calling and function arguments repair: Automatically heals stringified JSON arguments inside OpenAI `tool_calls` and Anthropic `tool_use` blocks.
- Delimiter stack balancer: Tracks nested object and array depth to close truncated JSON structures upon EOF.
- Markdown and chatter stripping: Removes conversational preambles and markdown code fences without regex overhead.
- Unescaped quote normalization: Escapes double quotes inside nested string values.
- Schema type coercion: Automatically coerces string numbers, string booleans, and singletons to match target JSON Schema definitions.
- Automatic schema inference: Generates standard JSON Schemas directly from example JSON outputs.
- Lexical audit trail: Explains repairs with line and column numbers, applied rules, and actions.
- Multi-provider routing: Routes across OpenAI, Anthropic, and Ollama through path prefixes or `X-SchemaDoctor-Upstream` headers.
- Prometheus exporter: Native `/metrics` endpoint for Grafana and Prometheus monitoring.
- Low overhead: Operates in less than 15MB of resident memory with less than 0.004ms of added wire latency.
- Embedded studio: Includes an embedded React 19 dashboard with live monitoring, traffic simulation, diff analysis, and schema management.
- Pure Go binary: Uses `modernc.org/sqlite` for zero-CGO persistence across Linux, macOS, and Windows.

## Performance

Benchmarks executed across 50,000 iterations per failure pattern on an Intel Core i5 processor:

| Syntax Failure Mode | Throughput | Average Latency | Memory Allocs | Repaired Output Valid |
| :--- | :--- | :--- | :--- | :--- |
| Truncated Array (`max_tokens`) | 352,902 ops/sec | 2.834 µs | 42 allocs/op | Yes |
| Markdown Code Fences | 402,979 ops/sec | 2.482 µs | 38 allocs/op | Yes |
| Unescaped Internal Quotes | 433,472 ops/sec | 2.307 µs | 36 allocs/op | Yes |
| Dangling Key & Trailing Comma | 452,621 ops/sec | 2.209 µs | 34 allocs/op | Yes |
| Python Constants (`True`/`None`) | 347,977 ops/sec | 2.874 µs | 44 allocs/op | Yes |

## Installation

### From Source

Ensure Go 1.22+ and Node.js 20+ are installed:

```bash
git clone https://github.com/alexandrmotologa/schemadoctor.git
cd schemadoctor
make build
```

The compiled binary will be placed at `bin/schemadoctor`.

## Quick Start

### 1. Command-Line Repair Tool (`fix`)

Repair malformed JSON directly through standard input or from files:

```bash
# Repair a file
schemadoctor fix examples/broken_samples/truncated_array.json

# Pipe through stdin
echo '{"user": {"name": "Alex", "items": [1, 2, ' | schemadoctor fix

# Coerce types against a schema
schemadoctor fix --schema schemas/user_profile.json examples/broken_samples/python_literals.json

# Overwrite file in place
schemadoctor fix --in-place broken.json
```

### 2. Start Reverse Proxy & Studio (`serve`)

Start the reverse proxy pointing to your upstream LLM provider:

```bash
# Point to OpenAI
schemadoctor serve --upstream https://api.openai.com --port 8080 --studio-port 4141

# Point to local Ollama instance
schemadoctor serve --upstream http://localhost:11434 --port 8080 --studio-port 4141

# Run with built-in mock LLM server for local testing without an API key
schemadoctor serve --mock-upstream --port 8080 --studio-port 4141
```

Once running:
- Reverse proxy listens on `http://localhost:8080`
- Studio UI and management API open at `http://localhost:4141`

### 3. Connect Client Applications

Update your client application base URL to route through schemadoctor:

#### Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-openai-api-key"
)

# Even if max_tokens cuts the output in half, the response returns valid JSON
response = client.chat.completions.create(
    model="gpt-4o-mini",
    max_tokens=60,
    messages=[
        {"role": "user", "content": "List 10 software architectural patterns as JSON."}
    ],
    extra_headers={
        "X-SchemaDoctor-Target-Schema": "user_profile"  # optional schema coercion
    }
)

print(response.choices[0].message.content)
```

#### TypeScript (OpenAI SDK)

```typescript
import OpenAI from "openai";

const openai = new OpenAI({
  baseURL: "http://localhost:8080/v1",
  apiKey: process.env.OPENAI_API_KEY,
});

const completion = await openai.chat.completions.create({
  model: "gpt-4o-mini",
  max_tokens: 50,
  messages: [{ role: "user", content: "Generate user account JSON" }],
});

console.log(JSON.parse(completion.choices[0].message.content!));
```

#### LangChain (Python)

```python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key",
    model="gpt-4o-mini"
)

result = llm.invoke("Generate team roster JSON")
print(result.content)
```

#### cURL (Streaming SSE)

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "max_tokens": 40,
    "messages": [{"role": "user", "content": "Generate a nested JSON product list"}]
  }'
```

## Configuration

schemadoctor can be configured using a YAML file or environment variables.

Example `schemadoctor.yaml`:

```yaml
server:
  proxy_port: 8080
  studio_port: 4141
  host: "0.0.0.0"

upstream:
  url: "https://api.openai.com"
  timeout_seconds: 60
  mock_upstream: false

database:
  path: "data/schemadoctor.db"

schemas:
  directory: "schemas"
```

Start the proxy using the config:

```bash
schemadoctor serve --config schemadoctor.yaml
```

Environment variables:
- `SCHEMADOCTOR_PORT`: Reverse proxy port (default: `8080`)
- `SCHEMADOCTOR_STUDIO_PORT`: Studio dashboard port (default: `4141`)
- `SCHEMADOCTOR_UPSTREAM_URL`: Target provider URL (default: `https://api.openai.com`)
- `SCHEMADOCTOR_MOCK_UPSTREAM`: Enable mock upstream (`true`/`false`)
- `SCHEMADOCTOR_DB_PATH`: Embedded SQLite database path

## Embedded React 19 Studio

schemadoctor bundles a self-contained React 19 dashboard served directly from memory by the Go binary. Open `http://localhost:4141` to inspect live traffic, benchmark AST repair latency, and test custom schemas.

### Live Monitor (Desktop)

Inspect real-time request volume, repair success rate, estimated token savings, and live LLM stream traces.

<p align="center">
  <img src="docs/images/dashboard_desktop.png?raw=true" alt="schemadoctor Desktop Live Monitor" width="100%" />
</p>

### Responsive Multi-Device Support (Tablet & Mobile)

The studio adapts to desktop, tablet, and mobile viewport sizes:

<p align="center">
  <img src="docs/images/dashboard_tablet.png?raw=true" alt="schemadoctor Tablet View" width="48%" />
  <img src="docs/images/dashboard_mobile.png?raw=true" alt="schemadoctor Mobile View" width="48%" />
</p>

### Interactive Playground & Lexical Audit Trail

Test damaged JSON snippets against candidate schemas and inspect line-by-line AST corrections with exact token offsets:

<p align="center">
  <img src="docs/images/playground_diff.png?raw=true" alt="schemadoctor Interactive Playground and Lexical Audit Trail" width="100%" />
</p>

## Documentation

- [Architecture Design](docs/ARCHITECTURE.md): Lexer design, delimiter stack mechanics, and SSE interception.
- [Benchmarks](docs/BENCHMARKS.md): Throughput, latency, and memory allocation profiles.
- [API Reference](docs/API_REFERENCE.md): REST endpoints, SSE streams, and CLI flags.

## License

This project is distributed under the MIT License. See [LICENSE](LICENSE) for details.
