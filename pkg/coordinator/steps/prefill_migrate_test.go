/*
Copyright 2026 The llm-d Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	reqcommon "github.com/llm-d/llm-d-router/pkg/common/request"

	"github.com/llm-d/llm-d-router/pkg/coordinator/config"
	"github.com/llm-d/llm-d-router/pkg/coordinator/gateway"
	"github.com/llm-d/llm-d-router/pkg/coordinator/pipeline"
)

// TestPrefillStep_ConnectFailureRetriesThenSucceeds verifies the prefill step
// retries a retryable upstream status on another pod when migration is enabled,
// excludes the failed pod, and returns the KV transfer params from the eventual
// success.
func TestPrefillStep_ConnectFailureRetriesThenSucceeds(t *testing.T) {
	var mu sync.Mutex
	var excludeHeaders []string
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		n := attempts
		excludeHeaders = append(excludeHeaders, r.Header.Get(reqcommon.HeaderExcludeEndpoints))
		mu.Unlock()

		w.Header().Set(reqcommon.HeaderDestinationEndpointServed, fmt.Sprintf("10.1.0.%d:8080", n))
		if n == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"kv_transfer_params": map[string]any{"block_id": "block-xyz", "peer_host": "10.1.0.2"},
		})
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewPrefillStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxConnectFailures:     2,
	})
	if err != nil {
		t.Fatal(err)
	}

	reqCtx := &pipeline.RequestContext{
		RequestID:        "req-prefill-mig",
		OriginalPath:     gateway.PathCompletions,
		Model:            "test-model",
		TokenIDs:         []int{1, 2, 3},
		Body:             map[string]any{"model": "test-model", "prompt": "Hello"},
		KVTransferParams: make(map[string]any),
	}

	if err := step.Execute(context.Background(), reqCtx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reqCtx.KVTransferParams["block_id"] != "block-xyz" {
		t.Fatalf("expected kv_transfer_params from the successful attempt, got %v", reqCtx.KVTransferParams)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if excludeHeaders[0] != "" {
		t.Errorf("first attempt should not exclude any pod, got %q", excludeHeaders[0])
	}
	if excludeHeaders[1] != "10.1.0.1:8080" {
		t.Errorf("retry should exclude the failed pod, got %q", excludeHeaders[1])
	}
}

// TestPrefillStep_ConnectFailureExhaustedReturnsError verifies the prefill step
// stops retrying once the budget is spent and surfaces the upstream error.
func TestPrefillStep_ConnectFailureExhaustedReturnsError(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewPrefillStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxConnectFailures:     1,
	})
	if err != nil {
		t.Fatal(err)
	}

	reqCtx := &pipeline.RequestContext{
		RequestID:        "req-prefill-fail",
		OriginalPath:     gateway.PathCompletions,
		Model:            "test-model",
		TokenIDs:         []int{1, 2, 3},
		Body:             map[string]any{"model": "test-model", "prompt": "Hello"},
		KVTransferParams: make(map[string]any),
	}

	if err := step.Execute(context.Background(), reqCtx); err == nil {
		t.Fatal("expected an error after retries are exhausted")
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Fatalf("expected 2 attempts (initial + 1 retry), got %d", attempts)
	}
}
