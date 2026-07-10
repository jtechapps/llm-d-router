# Precise vs. Approximate Prefix Caching: Resource Consumption Analysis

This report compares the compute and memory footprint of **Precise Prefix Caching** (using the `vllm-render` tokenizer sidecar running `Qwen/Qwen3-8B`) against **Approximate Prefix Caching** (`optimized-baseline` word-length matching) across input prompt sizes ranging from `1,000` to `100,000` tokens at a fixed request rate of **10 QPS**.

---

## 1. Executive Summary

- **Massive RAM Divergence at Large Contexts:** While approximate prefix caching requires negligible memory (`39 MiB` to `160 MiB`), precise prefix caching requires an ~1 GiB baseline RAM floor for Python/HF tokenizer vocabulary initialization, scaling up to **`9.48 GiB` (`9,485 MiB`)** at `100,000` prompt tokens — a **~59x memory increase**.
- **CPU Compute Overhead (~55% Increase at 100k Tokens):** Offloading exact tokenization over HTTP loopback to `vllm-render` requires **`1.11 cores` (`1110m`)** of Python compute at 100k tokens / 10 QPS. Total EPP pod CPU reaches **`2.30 cores`** compared to **`1.49 cores`** for approximate word-length matching.
- **Latency Parity for Normal Prompts ($\le 25\text{k}$):** For prompt sizes up to `25,000` tokens, end-to-end routing latency is virtually indistinguishable (`~0.94ms` to `1.02ms` P50). At `100,000` tokens, HTTP payload serialization and Python tokenization add `~0.67ms` to P50 latency (`1.57ms` vs. `0.90ms`).

---

## 2. Resource Consumption Comparison Table

All tests were executed at a fixed **10 QPS** against 10 backend simulator pods (`Qwen/Qwen3-8B`, `--max-model-len=131072`).

| Input Tokens | Metric | Approximate Caching (`optimized-baseline`) | Precise EPP Container (`epp`) | Precise Tokenizer (`vllm-render`) | Precise Total Pod (`TOTAL`) | Delta (Precise vs. Approx) |
|---|---|---|---|---|---|---|
| **1,000** | Peak CPU<br>Peak RAM<br>P50 Latency | `871m`<br>`39 MiB`<br>`0.95 ms` | `998m`<br>`42 MiB`<br>`0.94 ms` | `506m`<br>`975 MiB`<br>`-` | **`1,504m`**<br>**`1,017 MiB`**<br>**`0.94 ms`** | `+72.6% CPU`<br>`+2,507% RAM`<br>`-0.01 ms` |
| **5,000** | Peak CPU<br>Peak RAM<br>P50 Latency | `976m`<br>`46 MiB`<br>`0.96 ms` | `1,122m`<br>`52 MiB`<br>`0.99 ms` | `470m`<br>`1,043 MiB`<br>`-` | **`1,592m`**<br>**`1,095 MiB`**<br>**`0.99 ms`** | `+63.1% CPU`<br>`+2,280% RAM`<br>`+0.03 ms` |
| **10,000** | Peak CPU<br>Peak RAM<br>P50 Latency | `1,131m`<br>`48 MiB`<br>`0.95 ms` | `1,246m`<br>`60 MiB`<br>`1.01 ms` | `481m`<br>`1,054 MiB`<br>`-` | **`1,727m`**<br>**`1,114 MiB`**<br>**`1.01 ms`** | `+52.7% CPU`<br>`+2,220% RAM`<br>`+0.06 ms` |
| **15,000** | Peak CPU<br>Peak RAM<br>P50 Latency | `-`*<br>`-`*<br>`-`* | `1,266m`<br>`66 MiB`<br>`0.90 ms` | `722m`<br>`988 MiB`<br>`-` | **`1,988m`**<br>**`1,054 MiB`**<br>**`0.90 ms`** | `-` |
| **25,000** | Peak CPU<br>Peak RAM<br>P50 Latency | `1,147m`<br>`69 MiB`<br>`0.91 ms` | `1,167m`<br>`162 MiB`<br>`1.02 ms` | `1,036m`<br>`1,197 MiB`<br>`-` | **`2,203m`**<br>**`1,359 MiB`**<br>**`1.02 ms`** | `+92.1% CPU`<br>`+1,869% RAM`<br>`+0.11 ms` |
| **50,000** | Peak CPU<br>Peak RAM<br>P50 Latency | `1,400m`<br>`89 MiB`<br>`0.92 ms` | `1,003m`<br>`361 MiB`<br>`1.53 ms` | `1,088m`<br>`3,685 MiB`<br>`-` | **`2,091m`**<br>**`4,046 MiB`**<br>**`1.53 ms`** | `+49.4% CPU`<br>`+4,446% RAM`<br>`+0.61 ms` |
| **100,000** | Peak CPU<br>Peak RAM<br>P50 Latency | `1,488m`<br>`160 MiB`<br>`0.90 ms` | `1,192m`<br>`696 MiB`<br>`1.57 ms` | `1,110m`<br>`8,789 MiB`<br>`-` | **`2,302m`**<br>**`9,485 MiB`**<br>**`1.57 ms`** | `+54.7% CPU`<br>`+5,828% RAM`<br>`+0.67 ms` |

*\*Note: Step 15,000 was omitted from comparative delta calculations due to an anomalous 0ms recording in historical baseline logs.*

---

## 3. Deep-Dive Architectural Analysis

### A. Memory Footprint: Go String Parsing vs. Python Tokenizer Vocab
- **Approximate Caching (`optimized-baseline`):**
  - EPP executes entirely in compiled Go. It parses prompts using whitespace and heuristic byte ratios without loading vocabulary matrices or embedding tables.
  - At `100,000` tokens (~400 KB string payloads at 10 QPS = 4 MB/s ingress), Go GC string buffering accounts for the modest rise from `39 MiB` to `160 MiB`.
- **Precise Caching (`precise-baseline`):**
  - The `vllm-render` sidecar runs a Python FastAPI HTTP server (`vllm launch render`) that loads the HuggingFace `Qwen/Qwen3-8B` tokenizer vocabulary into RAM. This establishes an immediate **`~975 MiB` memory floor**.
  - As prompt sizes scale beyond `25,000` tokens, Python string allocation, JSON deserialization over loopback HTTP, and BPE tokenization arrays cause rapid memory accumulation. At `100,000` prompt tokens, Python object overhead and garbage collection hysteresis push container memory to **`8,789 MiB` (~8.8 GiB)**.
  - Meanwhile, the standalone `epp` container memory rises to `696 MiB` due to storing exact token integer slices (`[][]uint32`) and radix tree prefix indexes for 100k-token blocks.

### B. CPU Scaling: In-Process vs. Loopback IPC
- For small prompt sizes (`1,000` to `10,000` tokens), the CPU overhead of precise tokenization is ~0.50 to 0.60 cores above approximate caching, primarily spent on loopback HTTP network serialization between `epp` and `vllm-render`.
- At `100,000` tokens, `vllm-render` consumes **`1.11 cores` (`1110m`)** processing BPE token encoding at 1,000,000 tokens/sec ingress (100k tokens $\times$ 10 QPS). Total EPP pod compute reaches `2.30 cores` (`2302m`), compared to `1.49 cores` (`1488m`) for approximate word-length caching.

---

## 4. Engineering Recommendations & Trade-Offs

1. **When to use Approximate Prefix Caching:**
   - Recommended for high-throughput deployments ($> 100\text{ QPS}$) or resource-constrained environments where an extra `~8.8 GiB` RAM per EPP replica is cost-prohibitive.
   - For long system prompts and standard RAG workloads, approximate word-length matching achieves comparable routing cache-hit ratios without incurring sidecar compute or serialization latency.
2. **When to use Precise Prefix Caching:**
   - Essential when routing across heterogeneous model backends, multi-modal workloads, or complex speculative decoding pipelines where exact 16-token radix boundary alignment (`blockSizeTokens: 16`) is required to prevent KV cache fragmentation.
   - Deployments using precise prefix caching with long contexts ($\ge 100\text{k}$ tokens) should provision at least **`3.0 CPU cores` and `12 GiB RAM`** for the router pod to prevent OOM termination during traffic spikes.

---

## 5. Raw Benchmark Reference Links

- **Precise Prefix Cache Raw Results (Qwen3-8B):** [precise-baseline-1m-prefix-10qps.md](file:///usr/local/google/home/jacobmurry/inference/llmd/llm-d-router/llm-d-router-resource-tests/precise-baseline-1m-prefix-10qps.md)
- **Approximate Prefix Cache Raw Results:** [optimized-baseline-1m-prefix-10qps.md](file:///usr/local/google/home/jacobmurry/inference/llmd/llm-d-router/llm-d-router-resource-tests/optimized-baseline-1m-prefix-10qps.md)
- **Router Configuration Used:** [precise-baseline-1m-prefix.yaml](file:///usr/local/google/home/jacobmurry/inference/llmd/llm-d-router/scripts/perf-configs/router-configs/precise-baseline-1m-prefix.yaml)
- **Simulator Deployment Used:** [llm-d-sim-deployment-qwen3.yaml](file:///usr/local/google/home/jacobmurry/inference/llmd/llm-d-router/scripts/perf-configs/llm-d-sim-deployment-qwen3.yaml)
