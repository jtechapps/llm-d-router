# Precise Prefix Caching: Active-Active Scaling in Sidecar vs. Service Mode

This document presents an empirical comparison of **Precise Prefix Caching** (`Qwen/Qwen3-8B`, 100k input token ceiling, 10 QPS ingress load against 10 simulator replicas) with **Active-Active HA scaling across 3 EPP replicas**, evaluating both **Sidecar Mode** and **Service Mode**.

---

## 1. Executive Summary & Architectural Differences

To achieve true active-active load balancing where all replicated EPP pods actively serve traffic (rather than active-standby leader election gating where only 1 pod reports `Ready`), we explicitly configured `ha-enable-leader-election: false` in the container flags.

1. **Sidecar Mode (`router.proxy.mode: "sidecar"`):**
   - **Pod Structure:** Deploys **3 pods** total. Each pod encapsulates three containers: `epp`, `envoy-proxy`, and the `vllm-render` CPU tokenizer sidecar.
   - **Network Path:** Envoy communicates with EPP over internal loopback (`127.0.0.1:9002`).
2. **Service Mode (`router.proxy.mode: "service"`):**
   - **Pod Structure:** Deploys **6 pods** total: **3 standalone EPP pods** (each containing `epp` and `vllm-render`) and **3 standalone Envoy proxy pods** (`envoy-proxy`).
   - **Network Path:** Standalone Envoy proxies route ext_proc gRPC traffic across the Kubernetes network to the `llm-d-router-epp` Service FQDN.

---

## 2. Comprehensive Empirical Comparison Table

Both tests ran for 5 minutes at a steady client dispatch rate of 10 QPS (~1,000,000 prompt tokens/sec cluster ingress) against 10 identical simulation pods.

| Metric / Dimension | Sidecar Mode (`3 pods: epp + envoy + vllm`) | Service Mode (`3 EPP pods` + `3 Proxy pods`) | Delta / Analysis |
|---|---|---|---|
| **Achieved QPS & Goodput** | 10.0 QPS (0.0% error rate) | 10.0 QPS (0.0% error rate) | Identical throughput |
| **Total Pod Count** | **3 pods** | **6 pods** | De-coupled proxy deployments |
| **Peak Envoy Proxy CPU (per instance)** | ~0.95 cores (952m) | ~0.61 cores (609m) | Standalone proxies run leaner |
| **Peak EPP Core CPU (per instance)** | ~1.40 cores (1,401m) | ~1.70 cores (1,702m) | Service mode handles network gRPC ingress |
| **Peak Tokenizer CPU (per instance)** | ~0.68 cores (677m) | ~0.40 cores (395m) | Tokenization loop CPU variance |
| **Total Peak CPU (across all replicas)** | 3.03 cores × 3 pods = **~9.09 cores** | (2.10c × 3 EPP) + (0.61c × 3 Proxy) = **~8.13 cores** | Comparable total cluster compute (~8.1–9.1 cores) |
| **Peak Memory (per EPP pod)** | ~1.05 GiB (1,049 MiB) | ~1.06 GiB (1,084 MiB) | Dominated by vLLM string buffers (~1 GiB) |
| **Peak Memory (per Proxy pod)** | N/A (included in pod) | ~33 MiB | Standalone Envoy memory is minimal |
| **Total Cluster Memory** | **~3.15 GiB** | **~3.35 GiB** | +6% memory overhead for separate proxy pods |
| **P50 Routing Latency** | **1.35 ms** | **1.55 ms** | **+0.20 ms** (+14.8%) cross-pod gRPC overhead |
| **P95 Routing Latency** | **2.62 ms** | **4.13 ms** | **+1.51 ms** (+57.6%) cross-pod gRPC overhead |

---

## 3. Deep-Dive Analysis of Trade-Offs

### 1. Routing Latency: Loopback vs. Network Hop
- **Sidecar Mode** delivers superior tail latency (**P95 = 2.62 ms** vs. **4.13 ms** in Service Mode). Because Envoy runs in the same network namespace as EPP, ext_proc gRPC messages bypass network interface serialization and kube-proxy load balancing hops.
- **Service Mode** introduces ~1.5 ms of P95 latency overhead due to network transit and cross-pod gRPC connection multiplexing between the standalone Envoy pool and the EPP Service pool.

### 2. Resource Isolation & Compute Efficiency
- In **Sidecar Mode**, all three components (`epp`, `envoy-proxy`, `vllm-render`) share the same pod cgroup limits. Heavy tokenization CPU spikes on long 100k prompts can temporarily compete with Envoy proxy streaming threads within the same pod.
- In **Service Mode**, Envoy proxy pods operate in an isolated compute tier (~0.6 cores, ~33 MiB memory per pod). This allows Kubernetes schedulers to place lightweight proxy pods on edge nodes while scheduling memory-intensive EPP prefix indexer pods on high-capacity compute nodes.

---

## 4. Production Recommendations

1. **Default to Sidecar Mode for High-Performance Latency-Sensitive Routing:**
   When deploying precise prefix caching where minimizing TTFT (Time to First Token) overhead is critical, use **Sidecar Mode**. It saves cluster IP allocations, reduces pod overhead from 6 to 3 pods, and shaves **1.5 ms off P95 routing latency**.
2. **Use Service Mode for Independent Autoscaling & Edge Proxying:**
   If ingress connection counts scale independently from tokenization throughput (e.g., thousands of idle streaming client connections), deploy in **Service Mode**. This enables scaling the standalone Envoy proxy tier horizontally via HPA without duplicating ~1 GiB vLLM tokenizer sidecars across every proxy instance.
