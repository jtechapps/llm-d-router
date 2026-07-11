#!/usr/bin/env python3
import os
import re
import subprocess
import sys
import time

script_dir = os.path.dirname(os.path.abspath(__file__))
repo_root = os.path.abspath(os.path.join(script_dir, ".."))

shared_prefix_file = os.path.join(script_dir, "perf-configs", "shared_prefix_agentic-1m-1k.yaml")

configs = [
    ("random-only", os.path.join(script_dir, "perf-configs", "router-configs", "random-only.yaml"), "output-sweep-random-only"),
    ("random-passthrough", os.path.join(script_dir, "perf-configs", "router-configs", "random-passthrough.yaml"), "output-sweep-random-passthrough"),
]

output_lengths = [100, 1000, 5000, 10000]

def update_files(output_len, config_name):
    print(f"\n=======================================================")
    print(f"Updating configs for output_len={output_len} (config={config_name}, rate=10 QPS)")
    print(f"  system_prompt_len = 100")
    print(f"  question_len      = 100")
    print(f"  output_len        = {output_len}")
    print(f"=======================================================")
    
    with open(shared_prefix_file, "r") as f:
        sp_content = f.read()
    
    sp_content = re.sub(r'system_prompt_len:\s*\d+', 'system_prompt_len: 100', sp_content)
    sp_content = re.sub(r'question_len:\s*\d+', 'question_len: 100', sp_content)
    sp_content = re.sub(r'output_len:\s*\d+', f'output_len: {output_len}', sp_content)
    sp_content = re.sub(r'rate:\s*[\d.]+\s*\n(\s*)duration:\s*300', f'rate: 10\n\\1duration: 300', sp_content)
    
    with open(shared_prefix_file, "w") as f:
        f.write(sp_content)

def main():
    results_dir = os.path.join(repo_root, "llm-d-router-resource-tests")
    os.makedirs(results_dir, exist_ok=True)
    
    total_runs = len(configs) * len(output_lengths)
    current_run = 0
    
    for config_name, router_config_path, test_name in configs:
        results_file = os.path.join(results_dir, f"{test_name}.md")
        
        for out_len in output_lengths:
            current_run += 1
            print(f"\n>>> Starting Output-Token run {current_run}/{total_runs}: {config_name} at output_len={out_len} <<<")
            update_files(out_len, config_name)
            
            with open(results_file, "a") as f:
                f.write(f"\n<!-- Run for Output Tokens: {out_len} (input=200, rate=10 QPS) -->\n")
                
            cmd = [
                "python3", os.path.join(script_dir, "run_nightly_perf.py"),
                "--sim-deploy", os.path.join(script_dir, "perf-configs", "llm-d-sim-deployment-1m-context.yaml"),
                "--router-config", router_config_path,
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
                print(f"ERROR: Benchmark failed for {config_name} at output_len={out_len}", file=sys.stderr)
                with open(results_file, "a") as f:
                    f.write(f"<!-- FAILED for Output Tokens: {out_len} -->\n")
            else:
                print(f"SUCCESS: Benchmark completed for {config_name} at output_len={out_len}")
                
            time.sleep(10)

    print("\nAll Output-Token sweep runs completed!")

if __name__ == "__main__":
    main()
