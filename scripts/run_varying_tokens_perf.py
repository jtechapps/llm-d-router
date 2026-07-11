#!/usr/bin/env python3
import subprocess
import os
import shutil
import time
import re
import sys

token_sizes = [1000, 5000, 10000, 15000, 25000, 50000, 100000, 200000, 500000, 1000000]

shared_prefix_path = "scripts/perf-configs/shared_prefix_agentic-1m-1k.yaml"
router_config_path = "scripts/perf-configs/router-configs/optimized-baseline-1m-prefix.yaml"

# Backup
shared_prefix_backup = shared_prefix_path + ".bak"
router_config_backup = router_config_path + ".bak"

print(f"Backing up configs...")
shutil.copyfile(shared_prefix_path, shared_prefix_backup)
shutil.copyfile(router_config_path, router_config_backup)

def update_shared_prefix(file_path, system_prompt_len, question_len):
    with open(file_path, 'r') as f:
        content = f.read()
    content = re.sub(r'system_prompt_len:\s*\d+', f'system_prompt_len: {system_prompt_len}', content)
    content = re.sub(r'question_len:\s*\d+', f'question_len: {question_len}', content)
    with open(file_path, 'w') as f:
        f.write(content)

def update_router_config(file_path, max_prefix_tokens):
    with open(file_path, 'r') as f:
        content = f.read()
    content = re.sub(r'maxPrefixTokensToMatch:\s*\d+', f'maxPrefixTokensToMatch: {max_prefix_tokens}', content)
    with open(file_path, 'w') as f:
        f.write(content)

try:
    for size in token_sizes:
        print(f"\n==================================================")
        print(f"Starting test for token size: {size}")
        print(f"==================================================")
        
        # Calculate lengths based on user feedback:
        # For 1000 tokens: question_len=500, system_prompt_len=500.
        # For 5000 tokens: question_len=500, system_prompt_len=4500.
        # For >5000 tokens: question_len=5000, system_prompt_len=input_tokens_size-5000.
        if size == 1000:
            question_len = 500
            system_prompt_len = 500
        elif size == 5000:
            question_len = 500
            system_prompt_len = 4500
        else:
            question_len = 5000
            system_prompt_len = size - 5000
            
        update_shared_prefix(shared_prefix_path, system_prompt_len, question_len)
        print(f"Updated {shared_prefix_path}: question_len={question_len}, system_prompt_len={system_prompt_len}")
        
        update_router_config(router_config_path, size)
        print(f"Updated {router_config_path}: maxPrefixTokensToMatch={size}")
        
        cmd = [
            "python3", "scripts/run_nightly_perf.py",
            "--sim-deploy", "scripts/perf-configs/llm-d-sim-deployment-1m-context.yaml",
            "--router-config", router_config_path,
            "--test-name", "optimized-baseline-1m-prefix",
            "--perf-job", shared_prefix_path,
            "--sim-replicas", "20",
            "--gcp-project", "jacobmurry-gke-dev",
            "--epp-cpu=10",
            "--epp-memory=20Gi"
        ]
        
        print(f"Running command: {' '.join(cmd)}")
        start_time = time.time()
        # We run this synchronously and let it print to stdout/stderr
        res = subprocess.run(cmd, capture_output=False, text=True)
        duration = time.time() - start_time
        print(f"Test for token size {size} finished with return code {res.returncode} in {duration:.2f} seconds")
        
        if res.returncode != 0:
            print(f"Error: Test for token size {size} failed. Aborting remaining tests.", file=sys.stderr)
            sys.exit(1)

except Exception as e:
    print(f"An error occurred: {e}", file=sys.stderr)
    sys.exit(1)

finally:
    # Restore backups
    print("Restoring backups...")
    if os.path.exists(shared_prefix_backup):
        shutil.copyfile(shared_prefix_backup, shared_prefix_path)
        os.remove(shared_prefix_backup)
    if os.path.exists(router_config_backup):
        shutil.copyfile(router_config_backup, router_config_path)
        os.remove(router_config_backup)
    print("Backups restored.")
