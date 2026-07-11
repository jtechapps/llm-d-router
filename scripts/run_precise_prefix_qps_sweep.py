#!/usr/bin/env python3
import os
import re
import subprocess
import sys
import time

script_dir = os.path.dirname(os.path.abspath(__file__))
repo_root = os.path.abspath(os.path.join(script_dir, ".."))

orig_prefix_file = os.path.join(script_dir, "perf-configs", "shared_prefix_agentic-1m-1k.yaml")
qps_job_file = os.path.join(script_dir, "perf-configs", "shared_prefix_agentic_precise_qps.yaml")
router_config = os.path.join(script_dir, "perf-configs", "router-configs", "precise-baseline-1m-prefix.yaml")
sim_deploy = os.path.join(script_dir, "perf-configs", "llm-d-sim-deployment.yaml")

rates = [1, 5, 10, 20, 50, 100]

def update_perf_job(rate):
    system_prompt_len = 95000
    question_len = 5000
    
    print(f"\n=======================================================")
    print(f"Updating shared_prefix_agentic_precise_qps.yaml for rate={rate} QPS")
    print(f"  system_prompt_len = {system_prompt_len}")
    print(f"  question_len      = {question_len}")
    print(f"  output_len        = 1000")
    print(f"  rate              = {rate} QPS")
    print(f"=======================================================")
    
    with open(orig_prefix_file, "r") as f:
        content = f.read()
        
    content = re.sub(r'system_prompt_len:\s*\d+', f'system_prompt_len: {system_prompt_len}', content)
    content = re.sub(r'question_len:\s*\d+', f'question_len: {question_len}', content)
    content = re.sub(r'output_len:\s*\d+', 'output_len: 1000', content)
    content = re.sub(r'model_name:\s*[^\n]+', 'model_name: Qwen/Qwen3-8B', content)
    content = re.sub(r'rate:\s*[\d.]+\s*\n(\s*)duration:\s*300', f'rate: {rate}\n\\1duration: 300', content)
    
    with open(qps_job_file, "w") as f:
        f.write(content)

def main():
    results_dir = os.path.join(repo_root, "llm-d-router-resource-tests")
    os.makedirs(results_dir, exist_ok=True)
    results_file = os.path.join(results_dir, "precise-baseline-100k-qps-sweep.md")
    
    total_runs = len(rates)
    
    for idx, rate in enumerate(rates, 1):
        print(f"\n>>> Starting Precise-Prefix QPS run {idx}/{total_runs} for rate={rate} QPS <<<")
        update_perf_job(rate)
        
        with open(results_file, "a") as f:
            f.write(f"\n<!-- Run {idx}/{total_runs} for Request Rate: {rate} QPS (100k input tokens) -->\n")
            
        cmd = [
            "python3", os.path.join(script_dir, "run_nightly_perf.py"),
            "--sim-deploy", sim_deploy,
            "--router-config", router_config,
            "--test-name", "precise-baseline-100k-qps-sweep",
            "--perf-job", qps_job_file,
            "--sim-replicas", "10",
            "--gcp-project", "jacobmurry-gke-dev",
            "--epp-cpu=20",
            "--epp-memory=20Gi"
        ]
        
        print(f"Executing: {' '.join(cmd)}")
        res = subprocess.run(cmd)
        if res.returncode != 0:
            print(f"ERROR: Benchmark failed for rate={rate} QPS", file=sys.stderr)
            with open(results_file, "a") as f:
                f.write(f"<!-- FAILED for Request Rate: {rate} QPS -->\n")
        else:
            print(f"SUCCESS: Benchmark completed for rate={rate} QPS")
            
        time.sleep(10)

    print("\nAll Precise-Prefix QPS sweep runs completed!")

if __name__ == "__main__":
    main()
