# Profiling the adapters package

This guide shows how to measure performance and memory in the adapters package and what to expect.

## Quick start

Run micro-benchmarks (requires Go 1.20+):

```
cd adapters
go test -bench=. -benchmem -run ^$ -count=5 -timeout=10m
```

Capture CPU and memory profiles for a specific benchmark:

```
cd adapters
go test -bench=BenchmarkAdapter_BasicFieldCopy -benchmem -run ^$ -cpuprofile cpu.prof -memprofile mem.prof -count=1
```

View profiles:

```
go tool pprof -http=:0 cpu.prof
```

## What we measured (Nov 2025)

On a modern Linux dev machine, baseline numbers (before optimizations) were not captured due to a test setup issue. After fixes and small optimizations, functional tests pass and coverage is >85%. You can regenerate fresh numbers locally using the commands above.

## Hot paths and findings

- adaptStruct and adaptField dominate CPU time in typical runs.
- AdditionalData unmarshaling used to do O(n) lowercasing scans; now precomputed lowercase maps avoid per-key loops.
- AdditionalData marshaling used to allocate a map even when empty; now lazily allocated to reduce allocs in fully-mapped copies.
- Metadata building is cached; WarmMetadata can preload frequently used types in services that need low tail latency.

## Low-risk optimization ideas

- Precompute and store field index paths in a compact slice; already done.
- Avoid reflect.Value.Interface on unneeded paths; currently required for converters; could consider typed converters later.
- Consider pooling json.Encoder/Decoder for very high-throughput AdditionalData work; measure before adopting.
- If certain fields are frequently absent, consider a small bool map pool (already added) and tune its size threshold.

## Higher-effort options (measure first)

- Code generation for static adapters for hot structs.
- Typed converter registry keyed by reflect.Type rather than field name.
- SIMD-friendly lowercase hashing for AdditionalData keys (only if profiling shows hot).

## Reproducing under race detector

```
cd adapters
go test -race -count=3
```


