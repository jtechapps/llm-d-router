#!/usr/bin/env python3
import os
import re
import subprocess
import sys
import time

script_dir = os.path.dirname(os.path.abspath(__file__))
repo_root = os.path.abspath(os.path.join(script_dir, ".."))

orig_prefix_file = os.path.join(script_dir, "perf-configs", "shared_prefix_agentic-1m-1k.yaml")
precise_prefix_file = os.path.join(script_dir, "perf-configs", "shared_prefix_agentic_precise.yaml")
router_config = os.path.join(script_dir, "perf-configs", "router-configs", "precise-baseline-1m-prefix.yaml")
sim_deploy = os.path.join(script_dir, "perf-configs", "llm-d-sim-deployment.yaml")

token_sizes = [1000, 5000, 10000, 15000, 25000, 50000, 100000]

def update_perf_job(token_size):
    question_len = 500 if token_size <= 1000 else 5000
    system_prompt_len = token_size - question_len
    
    print(f"\n=======================================================")
    print(f"Updating shared_prefix_agentic_precise.yaml for token_size={token_size}")
    print(f"  system_prompt_len = {system_prompt_len}")
    print(f"  question_len      = {question_len}")
    print(f"  output_len        = 1000")
    print(f"  rate              = 10 QPS")
    print(f"=======================================================")
    
    with open(orig_prefix_file, "r") as f:
        content = f.read()
        
    content = re.sub(r'system_prompt_len:\s*\d+', f'system_prompt_len: {system_prompt_len}', content)
    content = re.sub(r'question_len:\s*\d+', f'question_len: {question_len}', content)
    content = re.sub(r'output_len:\s*\d+', 'output_len: 1000', content)
    content = re.sub(r'model_name:\s*[^\n]+', 'model_name: Qwen/Qwen3-8B', content)
    content = re.sub(r'rate:\s*[\d.]+\s*\n(\s*)duration:\s*300', 'rate: 10\n\\1duration: 300', content)
    
    with open(precise_prefix_file, "w") as f:
        f.write(content)

def main():
    results_dir = os.path.join(repo_root, "llm-d-router-resource-tests")
    os.makedirs(results_dir, exist_ok=True)
    results_file = os.path.join(results_dir, "precise-baseline-1m-prefix-10qps.md")
    
    total_runs = len(token_sizes)
    
    for idx, token_size in enumerate(token_sizes, 1):
        print(f"\n>>> Starting Precise-Prefix run {idx}/{total_runs} for token_size={token_size} <<<")
        update_perf_job(token_size)
        
        with open(results_file, "a") as f:
            f.write(f"\n<!-- Run {idx}/{total_runs} for Input Token Size: {token_size} -->\n")
            
        cmd = [
            "python3", os.path.join(script_dir, "run_nightly_perf.py"),
            "--sim-deploy", sim_deploy,
            "--router-config", router_config,
            "--test-name", "precise-baseline-1m-prefix-10qps",
            "--perf-job", precise_prefix_file,
            "--sim-replicas", "10",
            "--gcp-project", "jacobmurry-gke-dev",
            "--epp-cpu=20",
            "--epp-memory=20Gi"
        ]
        
        print(f"Executing: {' '.join(cmd)}")
        res = subprocess.run(cmd)
        if res.returncode != 0:
            print(f"ERROR: Benchmark failed for token_size={token_size}", file=sys.stderr)
            with open(results_file, "a") as f:
                f.write(f"<!-- FAILED for Input Token Size: {token_size} -->\n")
        else:
            print(f"SUCCESS: Benchmark completed for token_size={token_size}")
            
        time.sleep(10)

    print("\nAll Precise-Prefix token size sweep runs completed!")

if __name__ == "__main__":
    main()
