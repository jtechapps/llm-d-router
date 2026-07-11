# EPP Router Performance Benchmarking Results: random-passthrough-queuedepth-fixed

| Timestamp | Namespace | Router Config | Perf Job | Machine Family | Sim Replicas | EPP Images | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-10 06:01:36 | perf-qd-10 | random-passthrough-queuedepth | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 69 | 24 | 5209 | 95 | 0.82 | 1.94 | SUCCESS |
| 2026-07-10 06:01:36 | perf-qd-10 | random-passthrough-queuedepth | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 17 | 17 | 2805 | 52 | 0.82 | 1.94 | SUCCESS |
| 2026-07-10 06:01:36 | perf-qd-10 | random-passthrough-queuedepth | shared_prefix_job1.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | - | - | 2404 | 45 | 0.82 | 1.94 | SUCCESS |
