# EPP Router Performance Benchmarking Results: optimized-baseline-1m-prefix-active-request

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-07 23:13:19 | llm-d-perf-1783465999 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 221 | 40 | 9900 | 9655 | 1.82 | 9.07 | SUCCESS |
| 2026-07-07 23:13:19 | llm-d-perf-1783465999 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 26 | 17 | 2069 | 130 | 1.82 | 9.07 | SUCCESS |
| 2026-07-07 23:13:19 | llm-d-perf-1783465999 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 195 | 23 | 7852 | 9542 | 1.82 | 9.07 | SUCCESS |
