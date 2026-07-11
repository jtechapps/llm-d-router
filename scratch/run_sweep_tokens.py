import re
import subprocess
import os
import sys
import time
from datetime import datetime

sizes = [1000, 5000, 10000, 15000, 25000, 50000, 100000, 200000, 500000, 1000000]

job_config_path = "scripts/perf-configs/shared_prefix_agentic-1m-1k.yaml"
router_config_path = "scripts/perf-configs/router-configs/optimized-baseline-1m-prefix.yaml"
results_file = "llm-d-router-resource-tests/optimized-baseline-1m-prefix.md"
summary_file = "llm-d-router-resource-tests/sweep_summary.md"

def update_job_config(size):
    question_len = min(5000, int(size * 0.1))
    system_prompt_len = size - question_len
    
    with open(job_config_path, "r") as f:
        content = f.read()
        
    content = re.sub(r"system_prompt_len:\s*\d+", f"system_prompt_len: {system_prompt_len}", content)
    content = re.sub(r"question_len:\s*\d+", f"question_len: {question_len}", content)
    
    with open(job_config_path, "w") as f:
        f.write(content)
    print(f"[{datetime.now()}] Updated job config: system_prompt_len={system_prompt_len}, question_len={question_len}")

def update_router_config(size):
    with open(router_config_path, "r") as f:
        content = f.read()
        
    content = re.sub(r"maxPrefixTokensToMatch:\s*\d+", f"maxPrefixTokensToMatch: {size}", content)
    
    with open(router_config_path, "w") as f:
        f.write(content)
    print(f"[{datetime.now()}] Updated router config: maxPrefixTokensToMatch={size}")

def run_benchmark():
    cmd = [
        "python3", "scripts/run_nightly_perf.py",
        "--sim-deploy", "scripts/perf-configs/llm-d-sim-deployment-1m-context.yaml",
        "--router-config", router_config_path,
        "--test-name", "optimized-baseline-1m-prefix",
        "--perf-job", job_config_path,
        "--sim-replicas", "10",
        "--gcp-project", "jacobmurry-gke-dev",
        "--epp-cpu=20",
        "--epp-memory=20Gi"
    ]
    print(f"[{datetime.now()}] Running: {' '.join(cmd)}")
    
    # Run and log to stdout
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    for line in process.stdout:
        sys.stdout.write(line)
        sys.stdout.flush()
    process.wait()
    if process.returncode != 0:
        print(f"[{datetime.now()}] Benchmark failed with exit code {process.returncode}")
        return False
    return True

def parse_last_results():
    if not os.path.exists(results_file):
        return []
    with open(results_file, "r") as f:
        lines = f.readlines()
    rows = []
    # Loop backwards to find the last 3 data rows
    for line in reversed(lines):
        line = line.strip()
        if line.startswith("|") and not line.startswith("| Timestamp") and not line.startswith("|---"):
            rows.append(line)
            if len(rows) == 3:
                break
    return list(reversed(rows))

def main():
    sweep_results = []
    
    # Initialize summary file
    os.makedirs(os.path.dirname(summary_file), exist_ok=True)
    with open(summary_file, "w") as f:
        f.write("# EPP Token Size Sweep Summary\n\n")
        f.write("| Token Size | Container | Idle CPU (m) | Idle Mem (MiB) | Peak CPU (m) | Peak Mem (MiB) | P50 Latency (ms) | P95 Latency (ms) | Status |\n")
        f.write("|---|---|---|---|---|---|---|---|---|\n")

    for size in sizes:
        print(f"\n==================================================")
        print(f"Starting sweep run for size: {size}")
        print(f"==================================================")
        update_job_config(size)
        update_router_config(size)
        
        success = run_benchmark()
        if not success:
            print(f"Sweep stopped due to failure at size {size}")
            break
            
        rows = parse_last_results()
        print(f"Parsed last results: {rows}")
        
        # Append to summary file immediately
        with open(summary_file, "a") as f:
            for row in rows:
                parts = [p.strip() for p in row.split("|")[1:-1]]
                if len(parts) >= 15:
                    container = parts[7]
                    idle_cpu = parts[8]
                    idle_mem = parts[9]
                    peak_cpu = parts[10]
                    peak_mem = parts[11]
                    p50 = parts[12]
                    p95 = parts[13]
                    status = parts[14]
                    f.write(f"| {size} | {container} | {idle_cpu} | {idle_mem} | {peak_cpu} | {peak_mem} | {p50} | {p95} | {status} |\n")
                    f.flush()
                    
        # Sleep a bit to let GKE cool down/cleanup complete fully
        time.sleep(10)

    print(f"\nSweep completed! Summary written to {summary_file}")

if __name__ == "__main__":
    main()
