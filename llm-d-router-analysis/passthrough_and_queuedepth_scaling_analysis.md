# Random-Passthrough vs. Queue-Depth Router Scaling & Metric Scraping Analysis

This document provides a rigorous empirical comparison of three EPP routing configurations under idle cluster scaling and active load:
1. **`random-passthrough` (Baseline):** Default random endpoint selection with passthrough request handling and automatic background metric scraping.
2. **`random-passthrough-queuedepth`:** Random picker augmented with the `queue-scorer` plugin (weight 1), actively consuming `WaitingQueueSizeKey` from scraped backend pod metrics.
3. **`random-passthrough-noscrape`:** Identical to baseline, but with active HTTP `/metrics` background polling cleanly suppressed via `refresh-metrics-interval: "24h"`.

---

## 1. Executive Summary & Core Findings

1. **Automatic Metric Scraping Dominates Idle CPU Scaling:**
   In default passthrough configurations (`random-passthrough`), EPP automatically injects Prometheus metric scrapers (`metrics-data-source`) unless explicitly disabled. As backend cluster size grows from 10 to 100 simulator pods, EPP idle CPU consumption scales **linearly (+338% growth, from 192m to 649m)** due to active 50ms HTTP polling loops against all discovered endpoints.
2. **Disabling Metric Scraping Reduces 100-Replica Idle CPU by 87.1%:**
   Suppressing background metric collection in `random-passthrough-noscrape` keeps idle EPP CPU nearly flat across cluster scaling—consuming only **84m at 100 replicas** compared to **649m with scraping enabled**.
3. **Queue-Depth Scorer Impact Under Homogeneous Load:**
   Under a uniform fixed load (10 QPS / 100k input tokens across 10 identical simulation pods), `queue-scorer` does not alter active CPU/memory consumption (~5.2 cores total pod CPU, ~95 MiB RAM), but introduces minor routing latency overhead (P50 increases from `0.69 ms` to `0.82 ms`; P95 from `1.36 ms` to `1.94 ms`) due to scoring queue depth attributes across candidates on every request.

---

## 2. Idle Resource Usage Scaling Across Simulator Replicas

All idle measurements were captured over a 3-minute window across 10, 50, and 100 backend simulator pods (`llm-d-sim`).

| Router Configuration | Sim Replicas | Idle EPP CPU (m) | Idle Envoy CPU (m) | Total Idle Pod CPU (m) | Idle EPP Mem (MiB) | Idle Envoy Mem (MiB) | Total Idle Pod Mem (MiB) |
|---|---|---|---|---|---|---|---|
| **random-passthrough** | 10 | 192 | 25 | **217** | 21 | 17 | **38** |
| **random-passthrough** | 50 | 403 | 28 | **431** | 29 | 17 | **46** |
| **random-passthrough** | 100 | 649 | 35 | **684** | 37 | 17 | **54** |
| **random-passthrough-queuedepth** | 10 | 52 | 17 | **69** | - | 17 | **24** |
| **random-passthrough-queuedepth** | 50 | 300 | 19 | **319** | 29 | 17 | **46** |
| **random-passthrough-queuedepth** | 100 | 552 | 30 | **582** | 37 | 17 | **54** |
| **random-passthrough-noscrape** | 10 | 56 | 24 | **80** | 18 | 17 | **35** |
| **random-passthrough-noscrape** | 50 | 65 | 11 | **76** | 20 | 17 | **37** |
| **random-passthrough-noscrape** | 100 | 84 | 22 | **106** | 22 | 17 | **39** |

### Idle Scaling Analysis
- **Scraping Overhead:** In both `random-passthrough` and `random-passthrough-queuedepth`, each additional ~40 backend pods adds **~200m of continuous background CPU consumption** and **~8 MiB of memory**.
- **Noscrape Stability:** Without background polling, EPP memory grows by only **4 MiB** (from 18 MiB to 22 MiB) when scaling from 10 to 100 pods, representing pure Kubernetes endpoint discovery and watcher state.

---

## 3. Active Performance Under Fixed Load (10 QPS, 100k Input Tokens)

Active benchmark tests evaluated 5 minutes of steady 10 QPS client dispatch (~1,000,000 prompt tokens/sec ingress) against 10 simulator replicas.

| Router Configuration | Achieved QPS | Error Rate | Peak EPP CPU (m) | Peak Envoy CPU (m) | Peak Total CPU (m) | Peak Total Mem (MiB) | P50 Routing Latency (ms) | P95 Routing Latency (ms) |
|---|---|---|---|---|---|---|---|---|
| **random-passthrough** | 10.0 | 0.0% | 2,411 | 2,734 | **5,073** | 92 | **0.69** | **1.36** |
| **random-passthrough-noscrape** | 10.0 | 0.0% | 2,320 | 3,059 | **5,379** | 90 | **0.72** | **1.45** |
| **random-passthrough-queuedepth** | 10.0 | 0.0% | 2,404 | 2,805 | **5,209** | 95 | **0.82** | **1.94** |

### Active Load Analysis
- **Core Processing Dominance:** Under active ingress load, processing overhead is dominated by Envoy proxy streaming (~2.7–3.0 cores) and EPP request parsing/routing (~2.3–2.4 cores).
- **Queue-Scorer Overhead:** Evaluating queue depth across candidate endpoints adds ~0.13 ms to P50 latency and ~0.58 ms to P95 latency. In homogeneous simulator deployments where backend queue sizes remain balanced, this additional scoring calculation does not yield routing throughput gains.

---

## 4. Root Cause Analysis: Automatic Metric Scraping in EPP

1. **Automatic Injection in Loader:**
   In `pkg/epp/config/loader/defaults.go` (`ensureDataLayer`, lines 298–331), if `dataLayer.injectDefaults` is not explicitly set to `false`, EPP automatically injects `metrics-data-source` (Prometheus HTTP scraper) and `metrics-extractor`.
2. **Per-Endpoint Polling Goroutines:**
   When `NewEndpoint` is called in `pkg/epp/datalayer/runtime.go` (lines 369–403), a dedicated `Collector` goroutine is spawned for every discovered pod endpoint. Each collector polls `/metrics` on its target pod at `r.pollingInterval` (defaulting to `50ms`).
3. **Clean Opt-Out via Refresh Interval:**
   In pre-built EPP release images, omitting data sources when the data layer config object exists triggers a startup validation error (`data layer enabled but no data sources configured`). Setting `refresh-metrics-interval: "24h"` under `epp.flags` cleanly suppresses active polling loops without altering validation invariants or modifying pre-built container images.

---

## 5. Production Recommendations

1. **Explicitly Disable Scraping for Pure Passthrough Workloads:**
   For deployments using pure random or round-robin routing without queue-depth or KV-cache scoring plugins, configure `refresh-metrics-interval: "24h"` (or disable defaults in custom builds) to recover up to **~0.65 cores per 100 backend pods** in idle CPU capacity.
2. **Reserve Queue-Depth Scorer for Heterogeneous/Dynamic Latency Environments:**
   Deploy `queue-scorer` when backend serving pods experience variable token generation times or heterogeneous request lengths, where active queue balancing outweighs the ~0.58 ms P95 routing latency overhead.
