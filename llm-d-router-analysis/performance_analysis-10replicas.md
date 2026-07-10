# Performance Analysis: Impact of Input Token Size on EPP Router

This report analyzes the impact of input token size on the resource usage (CPU and Memory) and latency (P50 and P95) of the EPP Router, Envoy Proxy, and the total pod.

The benchmarks were run with token sizes ranging from 1,000 to 1,000,000 tokens.

## Executive Summary

- **CPU Usage**: Peak CPU usage for both EPP and Envoy Proxy shows a moderate increase as token size grows, but it is not strictly linear. EPP is the main consumer of CPU.
- **Memory Usage**: EPP peak memory usage shows a **strong positive correlation** with input token size, growing from ~30 MiB at 1k tokens to ~791 MiB at 1M tokens. Envoy Proxy memory usage remains low and stable (~20-34 MiB).
- **Latency**: P50 and P95 latencies remain extremely low and virtually unchanged across all token sizes (P50 constant at 0.05 ms, P95 varying between 0.10 ms and 0.15 ms). This indicates that the routing decision overhead is negligible even for very large contexts.

---

## Data Summary Table

The table below summarizes the idle and peak resource usage, along with latencies for each tested token size.

| Token Size | Container | Idle CPU (m) | Peak CPU (m) | Idle Mem (MiB) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) |
|---|---|---|---|---|---|---|---|
| **1,000** | **TOTAL** | **168** | **1,187** | **36** | **48** | **0.05** | **0.10** |
| | envoy-proxy | 13 | 382 | 14 | 19 | | |
| | epp | 155 | 805 | 22 | 30 | | |
| **5,000** | **TOTAL** | **190** | **1,322** | **37** | **53** | **0.05** | **0.13** |
| | envoy-proxy | 7 | 441 | 13 | 22 | | |
| | epp | 183 | 881 | 24 | 32 | | |
| **10,000** | **TOTAL** | **227** | **1,495** | **36** | **57** | **0.05** | **0.13** |
| | envoy-proxy | 15 | 532 | 13 | 23 | | |
| | epp | 212 | 963 | 23 | 35 | | |
| **15,000** | **TOTAL** | **219** | **1,283** | **38** | **62** | **0.05** | **0.10** |
| | envoy-proxy | 17 | 441 | 13 | 24 | | |
| | epp | 202 | 842 | 25 | 39 | | |
| **25,000** | **TOTAL** | **260** | **1,312** | **38** | **69** | **0.05** | **0.15** |
| | envoy-proxy | 16 | 430 | 13 | 25 | | |
| | epp | 244 | 896 | 25 | 45 | | |
| **50,000** | **TOTAL** | **250** | **1,377** | **37** | **94** | **0.05** | **0.10** |
| | envoy-proxy | 16 | 461 | 13 | 26 | | |
| | epp | 234 | 926 | 24 | 69 | | |
| **100,000** | **TOTAL** | **192** | **987** | **37** | **114** | **0.05** | **0.10** |
| | envoy-proxy | 21 | 293 | 13 | 28 | | |
| | epp | 171 | 712 | 24 | 87 | | |
| **200,000** | **TOTAL** | **165** | **1,252** | **37** | **153** | **0.05** | **0.11** |
| | envoy-proxy | 6 | 345 | 13 | 27 | | |
| | epp | 159 | 907 | 24 | 128 | | |
| **500,000** | **TOTAL** | **228** | **1,763** | **38** | **319** | **0.05** | **0.10** |
| | envoy-proxy | 8 | 559 | 14 | 30 | | |
| | epp | 220 | 1215 | 24 | 295 | | |
| **1,000,000** | **TOTAL** | **239** | **1,841** | **35** | **818** | **0.05** | **0.13** |
| | envoy-proxy | 18 | 473 | 13 | 34 | | |
| | epp | 221 | 1368 | 22 | 791 | | |

---

## Detailed Observations

### 1. CPU Usage Analysis

- **Idle vs. Peak**: Across all runs, Idle CPU for the EPP container hovered around **150m - 240m**, and Envoy Proxy was minimal (**6m - 21m**). Under load, Peak CPU increased significantly.
- **EPP Peak CPU**: Shows a general upward trend as token size increases. It starts at **805m** for 1k tokens, peaks at **1,368m** for 1M tokens. There are some non-monotonic behaviors (e.g., a drop to 712m at 100k tokens), which might be due to transient cluster conditions or scheduling variations.
- **Envoy Proxy Peak CPU**: Stays relatively stable between **293m and 559m**, with no strong correlation to token size. This suggests Envoy's routing/proxying overhead is mostly independent of the payload token size in this setup.

### 2. Memory Usage Analysis

- **Idle vs. Peak**: Idle memory was extremely stable for all components (EPP ~22-25 MiB, Envoy ~13-14 MiB).
- **EPP Peak Memory (Critical Trend)**: There is a clear, near-exponential growth in EPP peak memory usage as the token size increases:
  - 1k - 25k tokens: Memory stays under 50 MiB.
  - 50k tokens: 69 MiB.
  - 100k tokens: 87 MiB.
  - 200k tokens: 128 MiB.
  - 500k tokens: 295 MiB.
  - 1M tokens: **791 MiB** (a ~26x increase from 1k tokens).
  
  This is likely due to the EPP caching prefix information or maintaining state that scales with the `maxPrefixTokensToMatch` setting.
- **Envoy Proxy Peak Memory**: Remains very low and stable, increasing only slightly from **19 MiB** (1k tokens) to **34 MiB** (1M tokens).

### 3. Latency Analysis

- **P50 Latency**: Consistently **0.05 ms** across all token sizes.
- **P95 Latency**: Ranges from **0.10 ms to 0.15 ms** with no visible trend correlating to token size.
- **Conclusion**: The EPP routing decision latency is unaffected by the input token size. The overhead introduced by the router is negligible.

## Recommendations / Next Steps

1. **Memory Limits**: Since EPP memory scales significantly with token size, when deploying with large context windows (e.g., 1M tokens), the EPP memory limits must be configured accordingly. The current test used `--epp-memory=20Gi` (which sets limits to 40Gi), which is more than enough for 1M tokens (791 MiB peak), but for resource-constrained environments, a limit of 1.5 - 2 GiB would be safe for 1M tokens.
2. **Investigate Memory Growth**: We should investigate if the memory growth in EPP is due to caching and if there is a way to optimize it (e.g., TTLs, LRU eviction size) if it becomes a bottleneck.
