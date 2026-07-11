#!/usr/bin/env python3
import os
import re
import subprocess
import sys
import time

script_dir = os.path.dirname(os.path.abspath(__file__))
repo_root = os.path.abspath(os.path.join(script_dir, ".."))

shared_prefix_file = os.path.join(script_dir, "perf-configs", "shared_prefix_agentic-1m-1k.yaml")
sim_deploy_orig = os.path.join(script_dir, "perf-configs", "llm-d-sim-deployment-1m-context.yaml")
sim_deploy_high_itl = os.path.join(script_dir, "perf-configs", "llm-d-sim-deployment-high-itl.yaml")

configs = [
    ("random-only", os.path.join(script_dir, "perf-configs", "router-configs", "random-only.yaml"), "high-itl-random-only"),
    ("random-passthrough", os.path.join(script_dir, "perf-configs", "router-configs", "random-passthrough.yaml"), "high-itl-random-passthrough"),
]

rates = [10, 100, 200, 500]

def prepare_sim_deploy():
    with open(sim_deploy_orig, "r") as f:
        content = f.read()
    # Replace 0.1ms with 2.0ms (a 20x increase in generation latency)
    content = re.sub(r'--inter-token-latency\s*\n\s*0\.1ms', '--inter-token-latency\n        - 2.0ms', content)
    with open(sim_deploy_high_itl, "w") as f:
        f.write(content)
    print("Created high ITL sim deployment config with --inter-token-latency 2.0ms")

def update_files(rate, config_name):
    print(f"\n=======================================================")
    print(f"Updating configs for rate={rate} QPS (config={config_name}, High ITL=2.0ms)")
    print(f"  system_prompt_len = 100")
    print(f"  question_len      = 100")
    print(f"  output_len        = 100")
    print(f"=======================================================")
    
    with open(shared_prefix_file, "r") as f:
        sp_content = f.read()
    
    sp_content = re.sub(r'system_prompt_len:\s*\d+', 'system_prompt_len: 100', sp_content)
    sp_content = re.sub(r'question_len:\s*\d+', 'question_len: 100', sp_content)
    sp_content = re.sub(r'output_len:\s*\d+', 'output_len: 100', sp_content)
    sp_content = re.sub(r'rate:\s*[\d.]+\s*\n(\s*)duration:\s*300', f'rate: {rate}\n\\1duration: 300', sp_content)
    
    with open(shared_prefix_file, "w") as f:
        f.write(sp_content)

def main():
    prepare_sim_deploy()
    results_dir = os.path.join(repo_root, "llm-d-router-resource-tests")
    os.makedirs(results_dir, exist_ok=True)
    
    total_runs = len(configs) * len(rates)
    current_run = 0
    
    for config_name, router_config_path, test_name in configs:
        results_file = os.path.join(results_dir, f"{test_name}.md")
        
        for rate in rates:
            current_run += 1
            print(f"\n>>> Starting High-ITL run {current_run}/{total_runs}: {config_name} at {rate} QPS <<<")
            update_files(rate, config_name)
            
            with open(results_file, "a") as f:
                f.write(f"\n<!-- Run for Request Rate: {rate} QPS (input=200, output=100, ITL=2.0ms) -->\n")
                
            cmd = [
                "python3", os.path.join(script_dir, "run_nightly_perf.py"),
                "--sim-deploy", sim_deploy_high_itl,
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
                print(f"ERROR: Benchmark failed for {config_name} at {rate} QPS", file=sys.stderr)
                with open(results_file, "a") as f:
                    f.write(f"<!-- FAILED for Request Rate: {rate} QPS -->\n")
            else:
                print(f"SUCCESS: Benchmark completed for {config_name} at {rate} QPS")
                
            time.sleep(10)

    print("\nAll High-ITL QPS sweep runs completed!")

if __name__ == "__main__":
    main()
