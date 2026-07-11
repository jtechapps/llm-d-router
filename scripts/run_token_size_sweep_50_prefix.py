#!/usr/bin/env python3
import os
import re
import subprocess
import sys
import time

script_dir = os.path.dirname(os.path.abspath(__file__))
repo_root = os.path.abspath(os.path.join(script_dir, ".."))

shared_prefix_file = os.path.join(script_dir, "perf-configs", "shared_prefix_agentic-1m-1k.yaml")
router_config_file = os.path.join(script_dir, "perf-configs", "router-configs", "optimized-baseline-50-prefix.yaml")

token_sizes = [1000, 5000, 10000, 15000, 25000, 50000, 100000, 200000, 500000, 1000000]

def update_files(token_size):
    question_len = 500 if token_size <= 1000 else 5000
    system_prompt_len = token_size - question_len
    
    print(f"\n=======================================================")
    print(f"Updating configs for token_size={token_size}")
    print(f"  system_prompt_len = {system_prompt_len}")
    print(f"  question_len      = {question_len}")
    print(f"  maxPrefixTokensToMatch = 50 (FIXED)")
    print(f"=======================================================")
    
    with open(shared_prefix_file, "r") as f:
        sp_content = f.read()
    
    sp_content = re.sub(r'system_prompt_len:\s*\d+', f'system_prompt_len: {system_prompt_len}', sp_content)
    sp_content = re.sub(r'question_len:\s*\d+', f'question_len: {question_len}', sp_content)
    
    with open(shared_prefix_file, "w") as f:
        f.write(sp_content)

def main():
    test_name = "optimized-baseline-50-prefix-10qps"
    results_dir = os.path.join(repo_root, "llm-d-router-resource-tests")
    
    os.makedirs(results_dir, exist_ok=True)
    results_file = os.path.join(results_dir, f"{test_name}.md")
    
    for idx, token_size in enumerate(token_sizes, 1):
        print(f"\n>>> Starting run {idx}/{len(token_sizes)} for token_size={token_size} <<<")
        update_files(token_size)
        
        with open(results_file, "a") as f:
            f.write(f"\n<!-- Run {idx}/{len(token_sizes)} for Input Token Size: {token_size} -->\n")
            
        cmd = [
            "python3", os.path.join(script_dir, "run_nightly_perf.py"),
            "--sim-deploy", os.path.join(script_dir, "perf-configs", "llm-d-sim-deployment-1m-context.yaml"),
            "--router-config", router_config_file,
            "--test-name", test_name,
            "--perf-job", shared_prefix_file,
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

    print("\nAll 10 benchmark runs completed!")

if __name__ == "__main__":
    main()
