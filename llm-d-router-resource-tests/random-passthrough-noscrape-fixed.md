# EPP Router Performance Benchmarking Results: random-passthrough-noscrape-fixed

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-10 06:13:20 | perf-noscrape-10-v2 | random-passthrough-noscrape | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 80 | 35 | 5379 | 90 | 0.72 | 1.45 | SUCCESS |
| 2026-07-10 06:13:20 | perf-noscrape-10-v2 | random-passthrough-noscrape | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 24 | 17 | 3059 | 46 | 0.72 | 1.45 | SUCCESS |
| 2026-07-10 06:13:20 | perf-noscrape-10-v2 | random-passthrough-noscrape | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 56 | 18 | 2320 | 45 | 0.72 | 1.45 | SUCCESS |
