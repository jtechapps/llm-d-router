# EPP Router Performance Benchmarking Results: optimized-baseline-1m-prefix-rate-20-c3-150cpu-fixed-cache

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-08 02:41:15 | llm-d-perf-1783478475 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 164 | 44 | 11290 | 4459 | 1.22 | 2.18 | SUCCESS |
| 2026-07-08 02:41:15 | llm-d-perf-1783478475 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 13 | 17 | 1626 | 104 | 1.22 | 2.18 | SUCCESS |
| 2026-07-08 02:41:15 | llm-d-perf-1783478475 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 151 | 27 | 9664 | 4356 | 1.22 | 2.18 | SUCCESS |
