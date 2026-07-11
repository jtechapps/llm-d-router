# EPP Router Performance Benchmarking Results: optimized-baseline-1m-prefix-rate-20-c3-150cpu

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-08 02:20:18 | llm-d-perf-1783477218 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 205 | 43 | 11356 | 5993 | 1.20 | 3.07 | SUCCESS |
| 2026-07-08 02:20:18 | llm-d-perf-1783477218 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 28 | 17 | 1639 | 203 | 1.20 | 3.07 | SUCCESS |
| 2026-07-08 02:20:18 | llm-d-perf-1783477218 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 177 | 26 | 9717 | 5847 | 1.20 | 3.07 | SUCCESS |
