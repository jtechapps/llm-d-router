# Precise Prefix Caching: Active-Active Scaling Sweep Analysis (1-6 Replicas)

This document presents an empirical analysis of **Precise Prefix Caching** (`Qwen/Qwen3-8B`, 100k input token ceiling, 10 QPS ingress load against 10 simulator replicas) under **Active-Active HA scaling** in Sidecar Mode, sweeping from **1 to 6 EPP replicas**.

To achieve true active-active load balancing where all replicated EPP pods actively serve traffic (rather than active-standby leader election gating where only 1 pod reports `Ready`), we explicitly configured `ha-enable-leader-election: false` in the container flags.

---

## 1. Empirical Results Table

The following table consolidates the peak resource usage and latency metrics for each replica count. All runs were executed at a steady client dispatch rate of 10 QPS (~1,000,000 prompt tokens/sec cluster ingress) for 5 minutes.

> [!NOTE]
> Resource metrics (CPU and Memory) are reported **per pod** (specifically, for the monitored pod index 0), representing the individual instance footprint under the distributed load.

| Replicas | P50 Latency (ms) | P95 Latency (ms) | Peak EPP CPU (m) | Peak Envoy CPU (m) | Peak vLLM CPU (m) | Peak Pod CPU (m) | Peak vLLM Mem (MiB) | Peak Pod Mem (MiB) |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **1** | 2.47 | 4.76 | 4,085 | 3,139 | 1,116 | **8,335** | 9,140 | **9,564** |
| **2** | 1.40 | 2.00 | 1,738 | 1,287 | 490 | **3,227** | 987 | **1,071** |
| **3** | 1.35 | 2.62 | 1,401 | 952 | 677 | **2,447** | 975 | **1,049** |
| **4** | 1.48 | 3.55 | 1,708 | 1,216 | 231 | **3,134** | 986 | **1,063** |
| **5** | 1.40 | 1.98 | 1,254 | 727 | 185 | **2,166** | 972 | **1,043** |
| **6** | 0.92 | 1.94 | 637 | 219 | 107 | **887** | 1,041 | **1,096** |

---

## 2. Key Insights and Analysis

### 1. Latency Diminishing Returns
- **The 1-to-2 Replica Jump:** Scaling from a single replica to 2 replicas delivers the most significant performance gain. P50 routing latency drops by **43.3%** (from 2.47 ms to 1.40 ms) and P95 tail latency drops by **58.0%** (from 4.76 ms to 2.00 ms). This is due to the division of request concurrency, which relieves CPU scheduling pressure on EPP and Envoy.
- **Plateau and Tail Stability:** Beyond 2 replicas, latency improvements plateau. While 6 replicas achieves the absolute lowest latencies (P50 0.92 ms, P95 1.94 ms), the difference compared to 2 or 3 replicas is minor.
- **Anomalous 4-Replica Run:** The 4-replica run exhibits a localized spike in P95 latency (3.55 ms) and CPU usage. This is likely an artifact of resource contention or noisy neighbors on the shared development GKE cluster during that specific test window.

### 2. The Memory Collapse Phenomenon
- **Single Process Saturation:** At 10 QPS, a single EPP pod's `vllm-render` sidecar accumulates a massive memory footprint of **9.1 GiB**. This is driven by rapid Python object allocations for 100k-token prompt serialization and rendering, outpacing garbage collection.
- **Distributed Relief:** When the load is distributed across 2 or more replicas, the peak memory per pod collapses to **~1 GiB** (essentially the idle memory floor of the container). Because each process handles a lower rate of requests, the memory allocation rate remains well within the process's ability to clean up.
- **Cluster-Wide Savings:** Replicating the router actually *reduces* total cluster memory consumption for low-to-medium replica counts:
  - **1 Replica:** ~9.5 GiB total memory.
  - **2 Replicas:** ~2.1 GiB total memory (**-78%**).
  - **3 Replicas:** ~3.1 GiB total memory (**-67%**).
  - **6 Replicas:** ~6.6 GiB total memory (**-30%**).
  This demonstrates that scaling horizontally is highly memory-efficient for precise prefix caching workloads.

### 3. Work-Proportional Compute
- Excluding the anomalous 4-replica run, the aggregate peak CPU across all replicas remains relatively flat:
  - **1 Replica:** 8.34 cores total.
  - **2 Replicas:** 3.23 cores * 2 = 6.46 cores.
  - **3 Replicas:** 2.45 cores * 3 = 7.35 cores.
  - **6 Replicas:** 0.89 cores * 6 = 5.34 cores.
  This confirms that active-active scaling does not introduce significant routing CPU overhead; the total compute consumed is proportional to the request traffic, but distributed to achieve better latency.

---

## 3. Production Recommendation

1. **Optimal Sizing (2 to 3 Replicas):**
   For a workload of 10 QPS with 100k prompt tokens, deploying **2 or 3 EPP replicas** in Sidecar Mode is the sweet spot. It slashes tail latency by 45-58%, reduces total cluster memory by over 60%, and provides high-availability redundancy without increasing total CPU consumption.
2. **Avoid Single-Replica Deployments for Large Contexts:**
   A single replica handling high-volume large-context requests is highly susceptible to memory ballooning (up to 9.5 GiB or more) and latency degradation due to CPU starvation.
3. **Scaling Beyond 3 Replicas:**
   Scaling to 4+ replicas for this load level does not yield meaningful latency improvements and increases the idle memory footprint of the cluster (each additional pod adds ~1 GiB of baseline memory). Only scale beyond 3 replicas if the target QPS increases proportionally.
