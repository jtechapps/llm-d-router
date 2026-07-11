# EPP Router Performance Benchmarking Results: optimized-baseline-1m-prefix-rate-20-c3-150cpu-fixed-cache-gc-16cpu-sims

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-08 03:14:11 | llm-d-perf-1783480451 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 218 | 43 | 8395 | 5327 | 1.21 | 1.97 | SUCCESS |
| 2026-07-08 03:14:11 | llm-d-perf-1783480451 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 17 | 17 | 1338 | 84 | 1.21 | 1.97 | SUCCESS |
| 2026-07-08 03:14:11 | llm-d-perf-1783480451 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 201 | 26 | 7113 | 5248 | 1.21 | 1.97 | SUCCESS |
