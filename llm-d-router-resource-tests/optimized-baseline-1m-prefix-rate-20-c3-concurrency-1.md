# EPP Router Performance Benchmarking Results: optimized-baseline-1m-prefix-rate-20-c3-concurrency-1

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-08 01:48:59 | llm-d-perf-1783475339 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 190 | 42 | 6513 | 2132 | 1.02 | 1.97 | SUCCESS |
| 2026-07-08 01:48:59 | llm-d-perf-1783475339 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 15 | 17 | 822 | 73 | 1.02 | 1.97 | SUCCESS |
| 2026-07-08 01:48:59 | llm-d-perf-1783475339 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 175 | 25 | 5709 | 2077 | 1.02 | 1.97 | SUCCESS |
