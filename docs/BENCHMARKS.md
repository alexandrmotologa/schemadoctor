# Performance Benchmarks

This document records throughput and latency measurements for schemadoctor across common LLM JSON syntax failure modes.

## Environment

- Operating System: Windows 11 (x86_64)
- CPU: Intel Core i5-1135G7 @ 2.40GHz
- Go Version: go1.27.0
- Test Methodology: 50,000 iterations per failure pattern via `schemadoctor bench -n 50000`

## Results Summary

| Failure Pattern | Throughput | Average Latency | Memory Allocations | Output Valid |
| :--- | :--- | :--- | :--- | :--- |
| Truncated Array (`max_tokens`) | 352,902 ops/sec | 2.834 µs | 42 allocs/op | Yes |
| Markdown Fences (` ```json `) | 402,979 ops/sec | 2.482 µs | 38 allocs/op | Yes |
| Unescaped Internal Quotes | 433,472 ops/sec | 2.307 µs | 36 allocs/op | Yes |
| Dangling Key & Trailing Comma | 452,621 ops/sec | 2.209 µs | 34 allocs/op | Yes |
| Python Constants (`True`, `None`) | 347,977 ops/sec | 2.874 µs | 44 allocs/op | Yes |

## Latency Impact on Reverse Proxy Critical Path

The target latency budget for schemadoctor is under 1 millisecond (1,000 microseconds) of added processing time per request.

In production execution:
- Average AST repair time: 2.2 µs to 2.9 µs
- Added overhead: less than 0.003 ms per request
- The proxy consumes approximately 11MB of resident RAM during sustained traffic

## Running Benchmarks

Run the built-in benchmark command:

```bash
schemadoctor bench -n 100000
```

Or run Go internal benchmarks directly:

```bash
go test -bench=. -benchmem ./internal/ast
```
