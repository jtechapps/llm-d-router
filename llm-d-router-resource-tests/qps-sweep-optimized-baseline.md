
<!-- Run for Request Rate: 10 QPS (input=200, output=100) -->
| 2026-07-09 03:06:03 | llm-d-perf-1783566363 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 204 | 42 | 520 | 76 | 0.90 | 1.96 | SUCCESS |
| 2026-07-09 03:06:03 | llm-d-perf-1783566363 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 16 | 17 | 77 | 42 | 0.90 | 1.96 | SUCCESS |
| 2026-07-09 03:06:03 | llm-d-perf-1783566363 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 188 | 25 | 443 | 35 | 0.90 | 1.96 | SUCCESS |

<!-- Run for Request Rate: 100 QPS (input=200, output=100) -->
| 2026-07-09 03:14:43 | llm-d-perf-1783566883 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 208 | 44 | 208 | 48 | 0.00 | 0.00 | SUCCESS |
| 2026-07-09 03:14:43 | llm-d-perf-1783566883 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 33 | 17 | 33 | 19 | 0.00 | 0.00 | SUCCESS |
| 2026-07-09 03:14:43 | llm-d-perf-1783566883 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 175 | 27 | 175 | 29 | 0.00 | 0.00 | SUCCESS |

<!-- Run for Request Rate: 200 QPS (input=200, output=100) -->
| 2026-07-09 03:45:45 | llm-d-perf-1783568745 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 324 | 43 | 3206 | 95 | 1.50 | 4.72 | SUCCESS |
| 2026-07-09 03:45:45 | llm-d-perf-1783568745 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 29 | 17 | 825 | 56 | 1.50 | 4.72 | SUCCESS |
| 2026-07-09 03:45:45 | llm-d-perf-1783568745 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 295 | 26 | 2395 | 40 | 1.50 | 4.72 | SUCCESS |

<!-- Run for Request Rate: 500 QPS (input=200, output=100) -->
| 2026-07-09 03:54:13 | llm-d-perf-1783569253 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | TOTAL | 223 | 42 | 7017 | 112 | 3.59 | 78.55 | SUCCESS |
| 2026-07-09 03:54:13 | llm-d-perf-1783569253 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | envoy-proxy | 26 | 17 | 2080 | 65 | 3.59 | 78.55 | SUCCESS |
| 2026-07-09 03:54:13 | llm-d-perf-1783569253 | optimized-baseline-1m-prefix | shared_prefix_agentic-1m-1k.yaml | - | 10 | docker.io/envoyproxy/envoy:distroless-v1.33.2<br>ghcr.io/llm-d/llm-d-router-endpoint-picker-dev:main | epp | 197 | 25 | 4937 | 48 | 3.59 | 78.55 | SUCCESS |
