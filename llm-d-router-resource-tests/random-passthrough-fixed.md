# EPP Router Performance Benchmarking Results: random-passthrough-fixed

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-10 06:00:22 | perf-rp-fixed | random-passthrough | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 217 | 38 | 5073 | 92 | 0.69 | 1.36 | SUCCESS |
| 2026-07-10 06:00:22 | perf-rp-fixed | random-passthrough | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 25 | 17 | 2734 | 49 | 0.69 | 1.36 | SUCCESS |
| 2026-07-10 06:00:22 | perf-rp-fixed | random-passthrough | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 192 | 21 | 2411 | 45 | 0.69 | 1.36 | SUCCESS |
