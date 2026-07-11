# EPP Router Performance Benchmarking Results: optimized-baseline-1m-prefix-rate-20-speedup

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-08 00:26:14 | llm-d-perf-1783470374 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 192 | 42 | 8112 | 2839 | 0.00 | 0.00 | FAILED |
| 2026-07-08 00:26:14 | llm-d-perf-1783470374 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 29 | 17 | 1109 | 161 | 0.00 | 0.00 | FAILED |
| 2026-07-08 00:26:14 | llm-d-perf-1783470374 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 163 | 25 | 7049 | 2684 | 0.00 | 0.00 | FAILED |
