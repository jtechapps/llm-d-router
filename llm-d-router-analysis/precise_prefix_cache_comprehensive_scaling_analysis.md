# Comprehensive Scaling Analysis: Precise Tokenizer Prefix Caching (Qwen3-8B)

This report synthesizes empirical performance benchmarking data from two multi-step sweeps evaluating **Precise Tokenizer-Based Prefix Caching** in `llm-d-router` using `Qwen/Qwen3-8B` across the full stack (simulator pods, tokenizer sidecar container, and Endpoint Picker token producer).

1. **Input Token Size Sweep (Fixed 10 QPS):** Evaluates context scaling from `1,000` to `100,000` tokens.
2. **Request Rate (QPS) Sweep (Fixed 100,000 Tokens):** Evaluates throughput scaling from `1` to `20` QPS at large context lengths (up to 2,000,000 input tokens/sec ingress).

---

## Executive Summary

1. **Tokenization Compute Ceiling (`~1.2 Cores`):**
   Offloading BPE tokenization over HTTP loopback to `vllm-render` scales sub-linearly with request throughput. When scaling from `10 QPS` to `20 QPS` at 100,000 prompt tokens (doubling ingress from **1 million** to **2 million tokens/second**), `vllm-render` CPU consumption increases by just **5.8%** (from `1,119m` to `1,184m`). Rust-based HuggingFace tokenization kernels saturate 4 OpenMP worker threads efficiently without creating compute bottlenecks.

2. **Tokenizer RAM Saturation (`~15.3 GiB` at 20 QPS):**
   The Python `vllm-render` sidecar exhibits a memory floor of `~975 MiB` to hold the Fast API runtime and vocabulary matrices. Under high-load 100k token rendering, Python object allocation and garbage collection arenas expand dynamically with request concurrency, reaching **`15.3 GiB`** at `20 QPS`.

3. **Core Routing Compute Scales Linearly (`~13.6 Cores` at 20 QPS):**
   Unlike tokenization compute, the `envoy-proxy` and `epp` core routing containers scale linearly with request rate due to HTTP header parsing, prefix trie lookups, and endpoint scoring. At 20 QPS / 100k tokens, Envoy requires `6.17 cores` and EPP requires `6.79 cores`, bringing total EPP pod CPU to **`13.65 cores`**.

4. **Exceptional Latency Stability Under High Load:**
   End-to-end routing latency (including HTTP loopback tokenization rendering, prefix cache matching, and Envoy proxy forwarding) remains under **`3.0 ms` P50** and **`5.0 ms` P95** across all tested token sizes and request rates up to 20 QPS.

---

## Part 1: Input Token Size Scaling (Fixed 10 QPS)

This sweep measures how context length impacts resource consumption and routing latency at a constant 10 requests/second.

| Input Tokens | Envoy CPU (m) | EPP CPU (m) | vLLM-Render CPU (m) | Total Pod CPU (m) | vLLM-Render RAM (MiB) | Total Pod RAM (MiB) | P50 Latency (ms) | P95 Latency (ms) |
|---|---|---|---|---|---|---|---|---|
| **1,000** | 2,873 | 3,893 | 598 | **6,665** | 1,042 | **1,135** | 1.59 | 4.44 |
| **5,000** | 3,005 | 4,050 | 242 | **7,249** | 1,010 | **1,121** | 1.60 | 4.33 |
| **10,000** | 3,095 | 4,068 | 522 | **7,561** | 1,043 | **1,176** | 1.64 | 4.51 |
| **15,000** | 3,366 | 4,199 | 793 | **8,341** | 1,046 | **1,196** | 1.60 | 4.31 |
| **25,000** | 3,375 | 3,846 | 1,075 | **8,286** | 1,308 | **1,618** | 2.33 | 4.78 |
| **50,000** | 3,075 | 3,691 | 1,098 | **7,854** | 3,812 | **4,243** | 2.41 | 4.75 |
| **100,000** | 3,139 | 4,085 | 1,116 | **8,335** | 9,140 | **9,564** | 2.47 | 4.76 |

### Key Observations (Token Scaling)
- **RAM Expansion Transition:** Up to `15,000` tokens, total pod RAM is flat at `~1.1 GiB – 1.2 GiB`. Between `25,000` and `100,000` tokens, BPE integer array allocations over HTTP loopback trigger Python memory expansion, reaching `9.56 GiB` at 100k tokens.
- **Latency Step Function:** P50 latency holds steady at `~1.60 ms` for $\le 15\text{k}$ tokens, then transitions to a stable `~2.33 ms – 2.47 ms` plateau for $25\text{k} – 100\text{k}$ tokens as loopback JSON deserialization overhead increases.

---

## Part 2: Request Rate (QPS) Scaling (Fixed 100,000 Input Tokens)

This sweep evaluates stress testing by fixing context size at `100,000` prompt tokens and scaling request rate from `1 QPS` up to `20 QPS` (simulating 100k to 2M tokens/sec ingress).

| Request Rate (QPS) | Ingress Throughput (Tokens/sec) | Envoy CPU (m) | EPP CPU (m) | vLLM-Render CPU (m) | Total Pod CPU (m) | vLLM-Render RAM (MiB) | Total Pod RAM (MiB) | P50 Latency (ms) | P95 Latency (ms) |
|---|---|---|---|---|---|---|---|---|---|
| **1** | 100,000 | 386 | 1,097 | 722 | **1,972** | 1,125 | **1,300** | 0.99 | 1.93 |
| **5** | 500,000 | 1,662 | 2,358 | 1,099 | **5,040** | 4,265 | **4,734** | 1.91 | 4.68 |
| **10** | 1,000,000 | 3,267 | 3,934 | 1,119 | **8,305** | 8,934 | **9,617** | 2.83 | 4.81 |
| **20** | 2,000,000 | 6,170 | 6,786 | 1,184 | **13,649** | 15,301 | **16,720** | 2.69 | 4.87 |

### Key Observations (QPS Scaling)
- **Tokenization Efficiency:** The `vllm-render` sidecar demonstrates remarkable compute efficiency. At 1 QPS (100k tokens/sec), it consumes `722m` CPU. At 20 QPS (2M tokens/sec), it requires only `1,184m` CPU — processing **20x more tokens/sec with only 64% more CPU**.
- **Linear Memory & CPU Growth:** Core EPP routing compute scales linearly from `1.1 cores` at 1 QPS to `6.8 cores` at 20 QPS. Total container memory reaches `16.7 GiB` at 20 QPS as both python tokenization buffers and Go routing tries expand to handle 2 million prompt tokens/sec ingress.
- **Routing Latency Invariance:** Despite handling 2 million input tokens per second at 20 QPS, routing latency remains virtually unchanged (`2.69 ms` P50, `4.87 ms` P95 vs. `2.83 ms` P50 at 10 QPS).

---

## Architecture & Sizing Recommendations

1. **Resource Provisioning for Precise Caching Pods:**
   - **CPU Request / Limit:** For production deployments expecting high QPS at large context lengths (up to 20 QPS / 100k tokens), allocate at least **`8 cores` request / `16 cores` limit** per standalone EPP router pod (`6.8 cores` EPP + `6.2 cores` Envoy + `1.2 cores` vLLM-Render).
   - **Memory Request / Limit:** Set container memory limits to at least **`18 GiB`** when handling 100k context windows to accommodate the `~16.7 GiB` pod memory footprint at 20 QPS without triggering OOMKills.

2. **When to Choose Precise vs. Approximate Prefix Caching:**
   - Use **Precise Caching** when routing accuracy is paramount and pods are provisioned with $\ge 18\text{ GiB}$ RAM and $\ge 8\text{ cores}$.
   - Use **Approximate Caching** (`optimized-baseline`) in resource-constrained environments where RAM must remain under `500 MiB` per routing pod.
